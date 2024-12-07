package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"luxor.tech/tcp_msg_processing_test/internal/client"
	"luxor.tech/tcp_msg_processing_test/internal/server"
	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	rds_db "luxor.tech/tcp_msg_processing_test/rds-db"

	"github.com/stretchr/testify/require"
)

var once sync.Once
var testServer *server.Server

func InitServer() {
	once.Do(func() {
		err := logger.InitLogger("../config/log_config.json")
		if err != nil {
			panic("Failed to initialize logger: " + err.Error())
		}
		rds_db.GetDb()

		go func() {
			testServer = server.NewServer()
			err := testServer.Start(":8888")
			if err != nil {
				panic(err)
			}
		}()
	})
}

func TestServer(t *testing.T) {

	// start server
	InitServer()
	assert := require.New(t)

	username := "single-cli"
	serverAddr := "localhost:8888"
	c := client.NewClient(serverAddr, username, time.Second, time.Minute)
	err := c.Connect()
	assert.Nil(err)

	t.Run("not authorize", func(t *testing.T) {
		resp, err := c.Submit(1, "cliNonce", "calcuRes", true)
		assert.Nil(err)
		assert.NotNil(resp)
		assert.Equal(false, resp.Result)
		assert.True(strings.Contains(resp.Error, "Not authorized"))
	})

	t.Run("not exist job_id", func(t *testing.T) {
		err := c.Authorize()
		assert.Nil(err)
		resp, err := c.Submit(1000, "cliNonce", "calcuRes", true)
		assert.Nil(err)
		assert.NotNil(resp)
		assert.Equal(false, resp.Result)
		assert.True(strings.Contains(resp.Error, "Task does not exist"))
	})

	t.Run("duplicate clientNonce", func(t *testing.T) {
		err := c.Authorize()
		assert.Nil(err)
		testServer.DistributionToForTest(username) // distribute task
		job, _ := c.ReceiveRequest()
		tb, _ := json.Marshal(job.Params)
		var taskInfo client.Task
		_ = json.Unmarshal(tb, &taskInfo)
		jobID := taskInfo.JobID
		serverNonce := taskInfo.ServerNonce
		clientNonce, result := c.CalculateResult(serverNonce)
		_, _ = c.Submit(jobID, clientNonce, result, true)
		resp, err := c.Submit(jobID, clientNonce, result, true) // double
		assert.Nil(err)
		assert.NotNil(resp)
		assert.Equal(false, resp.Result)
		assert.True(strings.Contains(resp.Error, "Duplicate submission"))
	})

	t.Run("rate limit", func(t *testing.T) {
		err := c.Authorize()
		assert.Nil(err)
		// once
		testServer.DistributionToForTest(username) // distribute task
		job, _ := c.ReceiveRequest()
		tb, _ := json.Marshal(job.Params)
		var taskInfo client.Task
		_ = json.Unmarshal(tb, &taskInfo)
		jobID := taskInfo.JobID
		serverNonce := taskInfo.ServerNonce
		clientNonce, result := c.CalculateResult(serverNonce)
		_, _ = c.Submit(jobID, clientNonce, result, false)
		// twice
		testServer.DistributionToForTest(username) // distribute task
		job, _ = c.ReceiveRequest()
		tb, _ = json.Marshal(job.Params)
		_ = json.Unmarshal(tb, &taskInfo)
		jobID = taskInfo.JobID
		serverNonce = taskInfo.ServerNonce
		clientNonce, result = c.CalculateResult(serverNonce)
		resp, err := c.Submit(jobID, clientNonce, result, false) // double
		assert.Nil(err)
		assert.NotNil(resp)
		assert.Equal(false, resp.Result)
		assert.Equal("Submission too frequent", resp.Error)
	})

	t.Run("incorrect job_id", func(t *testing.T) {
		err := c.Authorize()
		assert.Nil(err)
		testServer.DistributionToForTest(username) // distribute task
		job, _ := c.ReceiveRequest()
		tb, _ := json.Marshal(job.Params)
		var taskInfo client.Task
		_ = json.Unmarshal(tb, &taskInfo)
		jobID := taskInfo.JobID
		serverNonce := taskInfo.ServerNonce
		clientNonce, result := c.CalculateResult(serverNonce)
		resp, err := c.Submit(jobID+1, clientNonce, result, true)
		assert.Nil(err)
		assert.NotNil(resp)
		assert.Equal(false, resp.Result)
		assert.True(strings.Contains(resp.Error, "Task does not exist"))
	})
}

func TestConcurrency(t *testing.T) {
	InitServer()
	serverAddr := "localhost:8888"
	assert := require.New(t)

	t.Run("Authorize concurrency", func(t *testing.T) {
		numClients := 100
		wg := sync.WaitGroup{}
		succCount := int64(0)
		for i := 0; i < numClients; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				username := fmt.Sprintf("user_%d", clientID)
				c := client.NewClient(serverAddr, username, time.Second, time.Minute)
				defer c.Close()
				_ = c.Connect()

				// Authorize client
				err := c.Authorize()
				if err != nil {
					t.Errorf("Client %d authorization failed: %v", clientID, err)
					return
				}
				atomic.AddInt64(&succCount, 1)
			}(i)
		}
		wg.Wait()
		assert.Equal(int64(numClients), succCount)
	})

	t.Run("Task concurrency", func(t *testing.T) {
		numClients := 100
		wg := sync.WaitGroup{}
		authorizeWg := sync.WaitGroup{}
		for clientID := 0; clientID < numClients; clientID++ {
			wg.Add(1)
			authorizeWg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				username := fmt.Sprintf("user_%d", clientID)
				c := client.NewClient(serverAddr, username, time.Second, time.Minute)
				defer c.Close()
				_ = c.Connect()

				err := c.Authorize()
				authorizeWg.Done()
				err = c.ReceiveTask(context.Background())
				if err != nil {
					t.Errorf("Client %d receive task failed: %v", clientID, err)
				}
			}(clientID)
		}
		// send tasks
		authorizeWg.Wait()
		testServer.StartTaskDistribution(time.Millisecond, 1)

		wg.Wait()
	})
}
