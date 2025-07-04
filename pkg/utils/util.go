package utils

import (
	"log"
	"os"
	"sync"
)

var appLogger *log.Logger
var once sync.Once

func SetupLogger() {
	once.Do(func() {
		appLogger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)
		appLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Println("Logger setup completed.")
	})
}

func Info(message string, args ...interface{}) {
	SetupLogger()
	appLogger.Printf("[INFO] "+message+" ", args...)
}

func Error(message string, args ...interface{}) {
	SetupLogger()
	appLogger.Printf("[ERROR] "+message+" ", args...)
}

func Debug(message string, args ...interface{}) {
	SetupLogger()
	appLogger.Printf("[DEBUG] "+message+" ", args...)
}
