package test

import (
	"log"
	"os"
	"time"
)

// Utility functions for debugging the language server.

var logFile *os.File
var outFile *os.File

func SetupLogging() {
	if logFile != nil {
		return
	}
	logFile, _ = os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	log.SetOutput(logFile)

	_, _ = logFile.Write(nil) // empty
}

// SetupDebugStdout reads from stdout and stderr and writes to a file.
//
// You can view stdout and stderr by reading a std.log file by using `tail -f ./std.log`.
func SetupDebugStdout() {
	if outFile != nil {
		return
	}
	outFile, _ = os.OpenFile("std.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	r, w, _ := os.Pipe()
	os.Stdout = w

	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	go func() {
		for {
			buff := make([]byte, 1024)
			_, _ = r.Read(buff)
			_, _ = outFile.Write(buff)

			buffErr := make([]byte, 1024)
			_, _ = rErr.Read(buffErr)
			_, _ = outFile.Write(buffErr)

			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// Log is a helper to log to a file during debugging or development.
//
// You can view logs by using the command `tail -f ./debug.log` in the root langauge server folder.
func Log(msg string) {
	SetupLogging()
	log.Println(msg)
}
