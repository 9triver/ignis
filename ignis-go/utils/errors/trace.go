//go:build stacktrace

package errors

import (
	goerrors "errors"
	"fmt"
	"runtime"
	"strings"
)

type unwrap interface {
	Unwrap() []error
}

type stacktrace interface {
	Stacktrace() string
}

type Error struct {
	curr      error
	callStack []string
}

func (e *Error) Stacktrace() string {
	sb := strings.Builder{}
	sb.WriteString("Error: ")
	sb.WriteString(e.curr.Error())
	sb.WriteString("\nStacktrace:\n")
	sb.WriteString(strings.Join(e.callStack, "\n"))
	return sb.String()
}

func (e *Error) Error() string {
	return e.curr.Error()
}

func New(text string) error {
	err := goerrors.New(text)

	pcs := make([]uintptr, 10)
	length := runtime.Callers(2, pcs)
	callStack := make([]string, 0, length)
	for _, pc := range pcs[:length] {
		f := runtime.FuncForPC(pc)
		file, lineno := f.FileLine(pc - 1)
		msg := fmt.Sprintf("%s:%d (0x%x)\n\t%s", file, lineno, pc, f.Name())
		callStack = append(callStack, msg)
	}

	return &Error{
		curr:      err,
		callStack: callStack,
	}
}

func Stacktrace(err error) string {
	if err, ok := err.(stacktrace); ok {
		return err.Stacktrace()
	}
	if errs, ok := err.(unwrap); ok {
		var trace []string
		for _, e := range errs.Unwrap() {
			trace = append(trace, Stacktrace(e))
		}
		return strings.Join(trace, "\ncaused by\n")
	}
	return err.Error()
}
