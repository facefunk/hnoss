package hnoss

import (
	"context"
	"log"
	"net/netip"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	mockTimeAdaptor struct {
		time time.Time
		err  error
		fun  func()
	}
	mockIPAdaptor struct {
		ip, putIP netip.Addr
		err       error
		called    bool
	}
	mockChatAdaptor struct {
		c                   chan string
		postChanID, postMsg string
		err                 error
	}
)

func (m *mockTimeAdaptor) Get() (time.Time, error) {
	return m.time, m.err
}

func (m *mockTimeAdaptor) Put(t time.Time) error {
	if m.fun != nil {
		m.fun()
	}
	return nil
}

func (m *mockIPAdaptor) Get() (netip.Addr, error) {
	m.called = true
	return m.ip, m.err
}

func (m *mockIPAdaptor) Put(ip netip.Addr) error {
	m.putIP = ip
	return nil
}

func (m *mockChatAdaptor) Chan() <-chan string {
	return m.c
}

func (m *mockChatAdaptor) Listen() error {
	return m.err
}

func (m *mockChatAdaptor) Close() error {
	return nil
}

func (m *mockChatAdaptor) Post(chanId, msg string) error {
	m.postChanID = chanId
	m.postMsg = msg
	return nil
}

var nextRunTimeTestCases = []struct {
	description                      string
	nowS, offsetS, intervalS, xNextS string
	xRunNow, xWasAdvanced            bool
	err                              error
}{
	{"RanNotFound", "2023-11-28T14:00:00Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T14:05:00Z", true, false, NewError("An error")},
	{"Regular", "2023-11-28T14:00:00Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T14:05:00Z", false, false, nil},
	{"Now", "2023-11-28T14:05:00Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T15:05:00Z", true, false, nil},
	{"Missed", "2023-11-28T14:00:00Z", "2023-11-27T14:06:00Z", "1h",
		"2023-11-28T14:06:00Z", true, false, nil},
	{"Advanced", "2023-11-28T14:00:00Z", "2023-11-27T14:04:00Z", "1h",
		"2023-11-28T15:04:00Z", false, true, nil},
	{"Long", "2023-05-12T14:00:00Z", "1977-05-25T11:00:00-07:00",
		strconv.Itoa(365*24) + "h", "2024-05-13T18:00:00Z", false, true, nil},
	{"Nano", "2023-11-28T14:00:00.1Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T14:05:00Z", false, false, nil},
}

func TestNext(t *testing.T) {
	logger := log.Default()
	r, err := time.Parse(time.RFC3339, "2023-11-28T13:05:00Z")
	require.NoError(t, err)
	ran := &mockTimeAdaptor{time: r}
	h := New(nil, logger, ran, nil, nil, nil, nil)

	for _, tc := range nextRunTimeTestCases {
		t.Run(tc.description, func(t *testing.T) {
			now := newTime(t, tc.nowS)
			offset := newTime(t, tc.offsetS)
			xNext := newTime(t, tc.xNextS)
			interval, err := time.ParseDuration(tc.intervalS)
			require.NoError(t, err)
			ran.err = tc.err
			next, jobNow, wasAdvanced := h.next(now, offset, interval)
			assert.Equal(t, tc.xRunNow, jobNow, "jobNow")
			assert.Equal(t, xNext, next, "next")
			assert.Equal(t, tc.xWasAdvanced, wasAdvanced, "wasAdvanced")
		})
	}
}

func TestScheduler(t *testing.T) {
	y := yamlConfig{
		Interval:        "1s",
		Offset:          "2023-11-28T00:00:00Z",
		IPMessageFormat: "%s",
	}
	conf := &Config{}
	err := conf.Set(&y)
	require.NoError(t, err)
	conf.Offset = time.Now().UTC()

	logger := log.Default()
	wg := &sync.WaitGroup{}
	ran := &mockTimeAdaptor{
		time: conf.Offset,
		fun: func() {
			wg.Done()
		},
	}
	ipService := &mockIPAdaptor{ip: newIP(t, "0.0.0.0")}
	ipCache := &mockIPAdaptor{ip: newIP(t, "1.2.3.4")}
	chat := &mockChatAdaptor{c: make(chan string)}
	now := NewRealNowAdapter()

	h := New(conf, logger, ran, ipService, ipCache, chat, now)
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		h.Start(ctx)
		wg.Done()
	}()

	wg.Wait()
	wg.Add(1)

	assert.Equal(t, "", chat.postChanID)
	assert.Equal(t, "0.0.0.0", chat.postMsg)

	chat.c <- "1234"
	chat.err = NewWarn("A warning")

	wg.Wait()
	wg.Add(1)

	assert.Equal(t, "1234", chat.postChanID)
	assert.Equal(t, "0.0.0.0", chat.postMsg)

	ipService.called = false
	ipCache.called = false
	chat.err = NewError("An error")

	wg.Wait()
	wg.Add(1)

	assert.False(t, ipService.called, "ipService")
	assert.False(t, ipCache.called, "ipCache")

	ran.fun = nil

	cancel()
	wg.Wait()
}

func TestGetIP(t *testing.T) {
	e := NewError("An error")

	ipService := &mockIPAdaptor{err: e}
	ipCache := &mockIPAdaptor{err: e}
	h := New(nil, nil, nil, ipService, ipCache, nil, nil)

	_, err := h.getIP(true)
	assert.Error(t, err)

	_, err = h.getIP(false)
	assert.Error(t, err)

	ipCache.err = nil
	ipCache.ip = newIP(t, "0.0.0.0")
	ip, err := h.getIP(true)
	assert.NoError(t, err)
	assert.Equal(t, ipCache.ip, ip)

	ipService.err = nil
	ipService.ip = newIP(t, "1.2.3.4")
	ip, err = h.getIP(false)
	assert.NoError(t, err)
	assert.Equal(t, ipService.ip, ip)
	assert.Equal(t, ipService.ip, ipCache.putIP)
}

func newTime(t *testing.T, s string) time.Time {
	n, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err)
	return n
}

func newIP(t *testing.T, s string) netip.Addr {
	ip, err := netip.ParseAddr(s)
	require.NoError(t, err)
	return ip
}

func TestLock(t *testing.T) {
	unlock, err := Lock("run/pid")
	require.NoError(t, err)
	err = unlock()
	assert.NoError(t, err)
}

func TestNewLogger(t *testing.T) {
	_, _, err := NewLogger("")
	assert.NoError(t, err)
	_, _, err = NewLogger("syslog")
	assert.NoError(t, err)
	_, closeLogFile, err := NewLogger("run/log")
	assert.NoError(t, err)
	err = closeLogFile()
	assert.NoError(t, err)
}

func TestPanicOnError(t *testing.T) {
	defer func() {
		a := recover()
		_, ok := a.(*Error)
		assert.True(t, ok)
	}()
	PanicOnError(nil)
	PanicOnError(func() error {
		return nil
	})
	PanicOnError(func() error {
		return NewError("An error")
	})
	assert.Fail(t, "didn't panic")
}
