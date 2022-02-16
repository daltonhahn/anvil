package logging

import (
	"os"
	"errors"
	"log"
)

var (
    WarningLogger *log.Logger
    InfoLogger    *log.Logger
    ErrorLogger   *log.Logger
)

func InitLog() {
	// Add a parameter for a user-defined path, otherwise, make the greedy log location in run_out next to current running term
	path := "run_out"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
    // If the file doesn't exist, create it or append to the file
    file, err := os.OpenFile(path+"/main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }
    InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime)
    WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime)
    ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger.Println("--------------------------------")
}
