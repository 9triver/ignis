package errors

import (
	goerrors "errors"
	"fmt"
)

func Join(errs ...error) error {
	return goerrors.Join(errs...)
}

func Unwrap(err error) error {
	return goerrors.Unwrap(err)
}

func As(err error, target any) bool {
	return goerrors.As(err, target)
}

func Is(err error, target error) bool {
	return goerrors.Is(err, target)
}

func Format(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return New(msg)
}

func WrapWith(err error, format string, args ...any) error {
	return Join(Format(format, args...), err)
}
