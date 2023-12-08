// Generated on 2023-12-08T01:54:22Z by gen.go
package hnoss

import (
	"fmt"

	"github.com/pkg/errors"
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
	return fmt.Sprintf("INFO: %s", s.error.Error())
}

func (s *Info) Unwrap() error {
	return s.error
}

func NewInfo(message string) *Info {
	return &Info{errors.New(message)}
}

func Infof(format string, args ...any) *Info {
	return &Info{errors.Errorf(format, args...)}
}

func InfoWrap(err error, message string) *Info {
	return &Info{errors.Wrap(err, message)}
}

func InfoWrapf(err error, format string, args ...any) *Info {
	return &Info{errors.Wrapf(err, format, args...)}
}

func (s *Warn) Error() string {
	return fmt.Sprintf("WARN: %s", s.error.Error())
}

func (s *Warn) Unwrap() error {
	return s.error
}

func NewWarn(message string) *Warn {
	return &Warn{errors.New(message)}
}

func Warnf(format string, args ...any) *Warn {
	return &Warn{errors.Errorf(format, args...)}
}

func WarnWrap(err error, message string) *Warn {
	return &Warn{errors.Wrap(err, message)}
}

func WarnWrapf(err error, format string, args ...any) *Warn {
	return &Warn{errors.Wrapf(err, format, args...)}
}

func (s *Error) Error() string {
	return fmt.Sprintf("ERROR: %s", s.error.Error())
}

func (s *Error) Unwrap() error {
	return s.error
}

func NewError(message string) *Error {
	return &Error{errors.New(message)}
}

func Errorf(format string, args ...any) *Error {
	return &Error{errors.Errorf(format, args...)}
}

func ErrorWrap(err error, message string) *Error {
	return &Error{errors.Wrap(err, message)}
}

func ErrorWrapf(err error, format string, args ...any) *Error {
	return &Error{errors.Wrapf(err, format, args...)}
}

func (s *Fatal) Error() string {
	return fmt.Sprintf("FATAL: %s", s.error.Error())
}

func (s *Fatal) Unwrap() error {
	return s.error
}

func NewFatal(message string) *Fatal {
	return &Fatal{errors.New(message)}
}

func Fatalf(format string, args ...any) *Fatal {
	return &Fatal{errors.Errorf(format, args...)}
}

func FatalWrap(err error, message string) *Fatal {
	return &Fatal{errors.Wrap(err, message)}
}

func FatalWrapf(err error, format string, args ...any) *Fatal {
	return &Fatal{errors.Wrapf(err, format, args...)}
}
