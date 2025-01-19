package util

import (
	"fmt"
	"time"
)

// EmptyLogger is used if no logger has been set up.
type EmptyLogger struct {
}

// log simply logs the given message to stdout if the message
// caller is allowed to log.
func (d *EmptyLogger) log(level, caller, msg string) {
	t := time.Now()
	fmt.Println(
		"["+level+"]",
		t.Format(time.RFC3339),
		caller,
		"▶ ",
		msg,
	)
}

// Debug is used to log debug messages.
func (d *EmptyLogger) Debug(caller, msg string) {
}

// Info is used to log info messages.
func (d *EmptyLogger) Info(caller, msg string) {
}

// Warning is used to log warning messages.
func (d *EmptyLogger) Warning(caller, msg string) {
	//d.log("WARNING", caller, msg)
}

// Error is used to log error messages.
func (d *EmptyLogger) Error(caller, msg string) {
	d.log("ERROR", caller, msg)
}

// Configure takes a configuration string separated by commas
// that contains all the callers that should be logged. This
// allows granular logging of different go files.
//
// Example:
//
//	logger.Configure("RootKey.go,Curve.go")
//	logger.Configure("all")
func (d *EmptyLogger) Configure(settings string) {
}
