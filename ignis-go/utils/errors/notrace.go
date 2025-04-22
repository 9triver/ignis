//go:build !stacktrace

package errors

import (
	goerrors "errors"
)

// no stacktrace is available, return errors.String as default
func Stacktrace(err error) string {
	return err.Error()
}

func New(text string) error {
	return goerrors.New(text)
}
