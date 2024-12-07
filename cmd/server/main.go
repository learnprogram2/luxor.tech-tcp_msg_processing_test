package main

import (
	"luxor.tech/tcp_msg_processing_test/internal/server"
	"luxor.tech/tcp_msg_processing_test/pkg/logger"
	rds_db "luxor.tech/tcp_msg_processing_test/rds-db"
)

// main 启动程序入口
func main() {
	// Initialize logger
	err := logger.InitLogger("config/log_config.json")
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Start db link
	rds_db.GetDb()

	newServer := server.NewServer()
	if newServer == nil {
		panic("create server nil")
	}
	err = newServer.Start(":8888")
	if err != nil {
		panic(err)
	}
}
