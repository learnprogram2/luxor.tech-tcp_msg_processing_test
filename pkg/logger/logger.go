package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Config struct {
	LogFile       string `json:"log_file"`
	LogLevel      string `json:"log_level"`
	LogFormat     string `json:"log_format"`
	LogTimeFormat string `json:"log_time_format"`
}

type Logger struct {
	logger *log.Logger
	level  string
}

// Global logger instance
var logger *Logger

// InitLogger initializes the logger based on the configuration file
func InitLogger(configFile string) error {
	// Read the configuration file
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := Config{}
	if err := decoder.Decode(&config); err != nil {
		return err
	}

	// Set output destination
	var output io.Writer
	if config.LogFile == "" {
		output = os.Stdout
	} else {
		outputFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		output = outputFile
	}

	logger = &Logger{
		logger: log.New(output, "", log.LstdFlags),
		level:  config.LogLevel,
	}

	if config.LogTimeFormat != "" {
		log.SetFlags(0)
		logger.logger.SetFlags(0)
		logger.logger.SetPrefix("")
		logger.logger.SetOutput(log.Writer())
	}

	return nil
}

func Info(str string, v ...interface{}) {
	logger.logger.Println("[INFO]", fmt.Sprintf(str, v...))
}
func Error(str string, v ...interface{}) {
	logger.logger.Println("[ERROR]", fmt.Sprintf(str, v...))
}
