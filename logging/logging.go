package logging

import (
	"os"
	"errors"
	"log"
)

var DefPath = "data/"

var (
    WarningLogger	*log.Logger
    InfoLogger		*log.Logger
    ErrorLogger		*log.Logger
	CatalogLogger	*log.Logger
	QuorumLogger	*log.Logger
)

func InitLog(log_path string) {
	DefPath = log_path
	// Add a parameter for a user-defined path, otherwise, make the greedy log location in run_out next to current running term
	if _, err := os.Stat(DefPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(DefPath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
    // If the file doesn't exist, create it or append to the file
    file, err := os.OpenFile(DefPath+"/main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }
    InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime)
    WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime)
    ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger.Println("--------------------------------")
}

func Spacer() string {
	return "----------------------------"
}

func CatalogLogInit() {
	// Add a parameter for a user-defined path, otherwise, make the greedy log location in run_out next to current running term
	if _, err := os.Stat(DefPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(DefPath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
    // If the file doesn't exist, create it or append to the file
    file, err := os.OpenFile(DefPath+"/catalog.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }

	CatalogLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime)
}

func QuorumLogInit() {
	// Add a parameter for a user-defined path, otherwise, make the greedy log location in run_out next to current running term
	if _, err := os.Stat(DefPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(DefPath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
    // If the file doesn't exist, create it or append to the file
    file, err := os.OpenFile(DefPath+"/quorum.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }

	QuorumLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime)
}

func ClearDefaultLogs() {
	if _, err := os.Stat(DefPath+"/main.log"); err == nil {
		err := os.Remove(DefPath+"/main.log")
		if err != nil {
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(DefPath+"/catalog.log"); err == nil {
		err := os.Remove(DefPath+"/catalog.log")
		if err != nil {
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(DefPath+"/quorum.log"); err == nil {
		err := os.Remove(DefPath+"/quorum.log")
		if err != nil {
			log.Fatal(err)
		}
	}
}
