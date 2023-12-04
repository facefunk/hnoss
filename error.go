package hnoss

import (
	gerrs "errors"
	"fmt"

	"github.com/pkg/errors"
)

const (
	InfoTitle  = "INFO"
	WarnTitle  = "WARN"
	ErrorTitle = "ERROR"
	FatalTitle = "FATAL"
)

type (
	Info struct {
		error
	}
	Warn struct {
		error
	}
	Error struct {
		error
	}
	Fatal struct {
		error
	}
)

func (s *Info) Error() string {
	return fmt.Sprintf("%s: %s", InfoTitle, s.error.Error())
}

func (s *Warn) Error() string {
	return fmt.Sprintf("%s: %s", WarnTitle, s.error.Error())
}

func (s *Error) Error() string {
	return fmt.Sprintf("%s: %s", ErrorTitle, s.error.Error())
}

func (s *Fatal) Error() string {
	return fmt.Sprintf("%s: %s", FatalTitle, s.error.Error())
}

func NewInfo(message string) *Info {
	return &Info{errors.New(message)}
}

func Infof(format string, args ...any) *Info {
	return &Info{errors.Errorf(format, args...)}
}

func ErrorWrap(err error, format string) *Error {
	return &Error{errors.Wrap(err, format)}
}

func FatalWrap(err error, format string) *Fatal {
	return &Fatal{errors.Wrap(err, format)}
}

func ErrorWrapf(err error, format string, args ...any) *Error {
	return &Error{errors.Wrapf(err, format, args...)}
}

func multiError(existErr *error, newErr error) bool {
	if newErr == nil {
		return false
	}
	if *existErr != nil {
		*existErr = gerrs.Join(*existErr, newErr)
		return true
	}
	*existErr = newErr
	return true
}
