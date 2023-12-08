package hnoss

import (
	gerrs "errors"
)

//go:generate go run gen.go

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
