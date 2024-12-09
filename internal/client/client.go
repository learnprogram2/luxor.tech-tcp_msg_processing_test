package client

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	"luxor.tech/tcp_msg_processing_test/pkg/util"
)

type Client struct {
	// session
	serverAddr string
	username   string
	conn       net.Conn

	// submission rate
	lastSubmit  time.Time
	minInterval time.Duration
	maxInterval time.Duration
}

func NewClient(serverAddr, username string, minInterval, maxInterval time.Duration) *Client {
	return &Client{
		serverAddr: serverAddr,
		username:   username,

		// submission rate
		minInterval: minInterval,
		maxInterval: maxInterval,
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	c.conn = conn
	logger.Info("Connected to server:%v", c.serverAddr)
	return nil
}

func (c *Client) Authorize() error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	authorizeRequest := Request{
		ID:     util.GenerateID(),
		Method: "authorize",
		Params: map[string]interface{}{
			"username": c.username,
		},
	}

	data, _ := json.Marshal(authorizeRequest)
	_, err := c.conn.Write(append(data, '\n'))
	if err != nil {
		logger.Error("Failed to send authorize request:%v", err)
		return fmt.Errorf("failed to send authorize request: %v", err)
	}
	response, err := c.ReadServerResponse()
	if err != nil {
		return err
	}
	if !response.Result {
		logger.Error("Authorization failed: %s", response.Error)
		return fmt.Errorf("authorization failed: %s", response.Error)
	}
	logger.Info("Authorization request succ:%s", c.username)
	return nil
}

func (c *Client) ReceiveTasks(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("Client stopped receiving tasks")
			return
		default:
			err := c.ReceiveTask(ctx)
			if err != nil {
				logger.Error("Failed to receive task: %v", err)
			}
		}
	}
}
func (c *Client) ReceiveTask(ctx context.Context) error {
	req, err := c.ReceiveRequest()
	if err != nil {
		return err
	}
	logger.Info("Received task: %v", req)
	// Handle the task
	if req.Method == "job" {
		var taskInfo Task
		tb, _ := json.Marshal(req.Params)
		_ = json.Unmarshal(tb, &taskInfo)

		jobID := taskInfo.JobID
		serverNonce := taskInfo.ServerNonce
		logger.Info("New task received: job_id=%d, server_nonce=%s", jobID, serverNonce)
		// task computation and submit the result
		clientNonce, result := c.CalculateResult(serverNonce)
		if result == "" {
			return fmt.Errorf("failed to calculate result")
		}

		response, err := c.Submit(jobID, clientNonce, result, true)
		if err != nil {
			return err
		}
		if response.Result {
			logger.Info("Result submitted successfully for job_id:%v, user:%s", jobID, c.username)
		} else {
			logger.Error("Result submission failed for job_id:%v", jobID, response.Error)
		}
	}

	return nil
}
func (c *Client) ReceiveRequest() (*Request, error) {
	reader := bufio.NewReader(c.conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		logger.Error("Error reading from server:%v", err)
		return nil, err
	}

	message = strings.TrimSpace(message)

	var task Request
	err = json.Unmarshal([]byte(message), &task)
	return &task, err
}

func (c *Client) CalculateResult(serverNonce string) (string, string) {
	// Ensure the client is connected
	if c.conn == nil {
		return "", ""
	}

	// Generate client_nonce
	clientNonce, err := GenerateClientNonce(16)
	if err != nil {
		logger.Error("Failed to generate client_nonce:", err)
		return "", ""
	}
	// Calculate SHA256(server_nonce + client_nonce)
	hashInput := serverNonce + clientNonce
	hash := sha256.Sum256([]byte(hashInput))
	result := hex.EncodeToString(hash[:])
	return clientNonce, result
}

func (c *Client) Submit(jobID int, clientNonce, result string, limit bool) (*Response, error) {
	// Prepare the submission request
	submitRequest := Request{
		ID:     util.GenerateID(),
		Method: "submit",
		Params: map[string]interface{}{
			"job_id":       jobID,
			"client_nonce": clientNonce,
			"result":       result,
		},
	}

	// Enforce submission rate
	if limit {
		if !c.lastSubmit.IsZero() {
			timeSinceLast := time.Now().Sub(c.lastSubmit)
			if timeSinceLast < c.minInterval {
				time.Sleep(c.minInterval - timeSinceLast)
			}
		}
	}
	c.lastSubmit = time.Now()

	// Serialize to JSON and send to server
	data, _ := json.Marshal(submitRequest)
	_, err := c.conn.Write(append(data, '\n'))
	if err != nil {
		logger.Error("Failed to send submission request:%v", err)
		return nil, err
	}

	return c.ReadServerResponse()
}

func GenerateClientNonce(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[num.Int64()]
	}
	return string(result), nil
}

func (c *Client) ReadServerResponse() (*Response, error) {
	reader := bufio.NewReader(c.conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	response = strings.TrimSpace(response)
	var result Response
	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON response: %v", err)
	}
	return &result, nil
}

func (c *Client) StartAutoSubmission(ctx context.Context) {
	ticker := time.NewTicker(c.maxInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping auto submission...")
			return
		case <-ticker.C:
			// Ensure at least one submission per minute
			if time.Since(c.lastSubmit) > time.Minute {
				_, _ = c.Submit(0, "", "", true) // job_id=0 for auto-submission
			}
		}
	}
}

// Close disconnects the client from the server
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		logger.Info("Disconnected from server")
	}
}
