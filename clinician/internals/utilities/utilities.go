package utilities

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger
var DB *sql.DB
var C context.Context
var appRoot string

func init() {
	exePath, err := os.Executable()
	if err != nil {
		appRoot = "."
	} else {
		appRoot = filepath.Dir(exePath)
	}

	logger = log.New(os.Stderr, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
}

func SetAppRoot(root string) {
	if root != "" {
		appRoot = root
	}
}

func AppRoot() string {
	return appRoot
}

func AppPath(parts ...string) string {
	segments := append([]string{appRoot}, parts...)
	return filepath.Join(segments...)
}

func SetLogFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	logger.SetOutput(file)
	return nil

}

// for logging
func Info(args ...interface{}) {
	logger.SetPrefix("INFO ")
	logger.Println(args...)
}

func Danger(args ...interface{}) {
	logger.SetPrefix("ERROR ")
	logger.Println(args...)
}

func Warning(args ...interface{}) {
	logger.SetPrefix("WARNING ")
	logger.Println(args...)
}
