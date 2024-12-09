package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"luxor.tech/tcp_msg_processing_test/internal/client"
	"luxor.tech/tcp_msg_processing_test/pkg/logger"
)

func main() {
	// Initialize logger
	err := logger.InitLogger("config/log_config_client.json")
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Server address and username
	serverAddr := "localhost:8888"
	username := "test_user" // todo generate username

	// Create a new client
	cli := client.NewClient(serverAddr, username, time.Second, time.Minute)

	// Connect to the server
	err = cli.Connect()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer cli.Close()

	// Authorize the client
	err = cli.Authorize()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// auto-submission: start a goroutine to submit solutions periodically within max interval
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cli.StartAutoSubmission(ctx)

	// Start deal tasks
	cli.ReceiveTasks(ctx)
}
