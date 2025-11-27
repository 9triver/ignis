package errorhandler

import (
	"fmt"
	"log"
	"runtime"
)

type ErrorAction int

const (
	LogOnly ErrorAction = iota
	LogAndPanic
	LogAndFatal
)

// ErrorHandler checks if an error is present and handles it according to the specified action.
// It wraps the error with the file and line number where the error occurred.
// The callDepth parameter specifies the number of call stack frames to skip when determining the error location.
//
// Example:
//
//	if err := someFunction(); ErrorHandler(err, 1, LogAndPanic) {
//	    // Handle the error case
//	}
func ErrorHandler(err error, callDepth int, action ErrorAction) bool {
	if err != nil {
		_, file, line, ok := runtime.Caller(callDepth)
		if !ok {
			panic("error when get caller")
		}
		err = fmt.Errorf("%s err at %d: %w", file, line, err)
		switch action {
		case LogOnly:
			log.Println(err)
		case LogAndPanic:
			log.Panic(err)
		case LogAndFatal:
			log.Fatal(err)
		}
		return true
	}
	return false
}
