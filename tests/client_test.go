package tests

import (
	"testing"
	"time"

	"luxor.tech/tcp_msg_processing_test/internal/client"
	"luxor.tech/tcp_msg_processing_test/internal/server"
	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	rds_db "luxor.tech/tcp_msg_processing_test/rds-db"

	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	Init()
	assert := require.New(t)

	// Initialize client
	c := client.NewClient("localhost:8888", "test_user", time.Second, time.Minute)
	defer c.Close()

	t.Run("authorize", func(t *testing.T) {
		assert.Nil(c.Connect())
		err := c.Authorize()
		assert.Nil(err)
	})
}

func TestUtils(t *testing.T) {
	nonce, err := client.GenerateClientNonce(11)
	assert := require.New(t)
	assert.Nil(err)
	assert.Equal(11, len(nonce))
}

func Init() {
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
