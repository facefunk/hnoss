package hnoss

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Remove run dir for idempotency.
	if err := os.RemoveAll("run"); err != nil {
		panic(err)
	}
	code := m.Run()
	if err := os.RemoveAll("run"); err != nil {
		panic(err)
	}
	os.Exit(code)
}

const (
	socket      = "45782"
	HTTPAddress = "http://localhost:" + socket + "/"
)

func serve() *http.Server {
	handler := http.FileServer(http.Dir("./testdata"))
	http.Handle("/", handler)
	server := &http.Server{Addr: ":" + socket, Handler: handler}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("failed to listen: %s", err))
		}
	}()

	return server
}

func shutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("failed to shutdown server: %s", err))
	}
}

func TestTextFileTimeAdapter(t *testing.T) {
	m := NewTextFileTimeAdapter("run/ran")
	ti, err := time.Parse(time.RFC3339, "2023-11-28T00:00:00Z")
	require.NoError(t, err)
	err = m.Put(ti)
	assert.NoError(t, err)
	ti2, err := m.Get()
	assert.NoError(t, err)
	assert.True(t, ti.Equal(ti2))
}

func TestTextFileIPAdapter(t *testing.T) {
	m := NewTextFileIPAdapter("run/ip")
	ip, err := netip.ParseAddr("1.2.3.4")
	require.NoError(t, err)
	err = m.Put(ip)
	assert.NoError(t, err)
	ip2, err := m.Get()
	assert.NoError(t, err)
	assert.Equal(t, ip, ip2)
}

func TestPlainTextIPServiceAdapter(t *testing.T) {
	server := serve()

	ip, err := netip.ParseAddr("1.2.3.4")
	require.NoError(t, err)

	m := NewPlainTextIPServiceAdapter("testdata/ip")
	ip2, err := m.Get()
	assert.NoError(t, err)
	assert.Equal(t, ip, ip2)

	m = NewPlainTextIPServiceAdapter("testdata/badip")
	ip2, err = m.Get()
	assert.Error(t, err)

	m = NewPlainTextIPServiceAdapter(HTTPAddress + "ip")
	ip2, err = m.Get()
	assert.NoError(t, err)
	assert.Equal(t, ip, ip2)

	m = NewPlainTextIPServiceAdapter(HTTPAddress + "badip")
	ip2, err = m.Get()
	assert.Error(t, err)

	shutdown(server)
}
