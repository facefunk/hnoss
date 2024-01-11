package hnoss

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
)

type (
	Logger struct {
		path   string
		logger *log.Logger
		close  func() error
	}
)

func NewLogger(path string) (*Logger, error) {
	l := &Logger{path: path}
	return l, l.connect()
}

func (l *Logger) connect() error {
	l.logger = nil
	l.close = nil
	switch l.path {
	case "":
		l.logger = log.Default()
		return nil
	case "syslog":
		ll, err := syslog.NewLogger(syslog.LOG_SYSLOG, log.LstdFlags)
		if err != nil {
			return FatalWrap(err, "failed to create syslog logger")
		}
		l.logger = ll
		return nil
	default:
		err := mkDir(l.path, "log")
		if err != nil {
			return (*Fatal)(err.(*Error))
		}
		file, err := os.OpenFile(l.path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return FatalWrap(err, "failed to open log file")
		}
		closeFile := closeFileFunc(l.path, "log", &err, file)
		l.logger = log.New(file, "", log.LstdFlags)
		l.close = func() error {
			closeFile()
			return err
		}
		return nil
	}
}

func (l *Logger) Log(e error) {
	if l.logger == nil {
		fmt.Fprint(os.Stderr, "failed to log: logger not set")
	}
	err := l.logger.Output(1, e.Error())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to log: %s\n", err)
		err = l.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to close log file for reopening: %s\n", err)
		}
		err = l.connect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect logger: %s\n", err)
			return
		}
		err = l.logger.Output(1, e.Error())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to log on 2nd attempt: %s\n", err)
		}
	}
}

func (l *Logger) Close() error {
	if l.close == nil {
		return nil
	}
	return l.close()
}
