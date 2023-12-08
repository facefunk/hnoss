package hnoss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiError(t *testing.T) {
	var err, newErr, existErr error
	b := multiError(&err, newErr)
	assert.False(t, b)
	newErr = NewError("new error")
	b = multiError(&err, newErr)
	assert.True(t, b)
	assert.Equal(t, newErr, err)
	existErr = NewError("existing error")
	err = existErr
	b = multiError(&err, newErr)
	assert.True(t, b)
	assert.NotEqual(t, existErr, err)
}
