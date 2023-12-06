package hnoss

import (
	"context"
	gerrs "errors"
	"log"
	"net/netip"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	mockTimeAdaptor struct {
		time, putTime time.Time
		err           error
	}
	mockIPAdaptor struct {
		ip, putIP netip.Addr
		err       error
	}
	mockChatAdaptor struct {
		c                   chan string
		postChanID, postMsg string
	}
	mockNowAdapter struct {
		now time.Time
	}
)

func (m *mockTimeAdaptor) Get() (time.Time, error) {
	return m.time, m.err
}

func (m *mockTimeAdaptor) Put(t time.Time) error {
	m.putTime = t
	return nil
}

func (m *mockIPAdaptor) Get() (netip.Addr, error) {
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
	return nil
}

func (m *mockChatAdaptor) Close() error {
	return nil
}

func (m *mockChatAdaptor) Post(chanId, msg string) error {
	m.postChanID = chanId
	m.postMsg = msg
	return nil
}

func (m *mockNowAdapter) Now() time.Time {
	return m.now
}

var nextRunTimeTestCases = []struct {
	description                      string
	nowS, offsetS, intervalS, xNextS string
	xRunNow, xWasAdvanced            bool
}{
	{"Regular", "2023-11-28T14:00:00Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T14:05:00Z", false, false},
	{"Now", "2023-11-28T14:05:00Z", "2023-11-27T14:05:00Z", "1h",
		"2023-11-28T15:05:00Z", true, false},
	{"Missed", "2023-11-28T14:00:00Z", "2023-11-27T14:06:00Z", "1h",
		"2023-11-28T14:06:00Z", true, false},
	{"Advanced", "2023-11-28T14:00:00Z", "2023-11-27T14:04:00Z", "1h",
		"2023-11-28T15:04:00Z", false, true},
	{"Long", "2023-05-12T14:00:00Z", "1977-05-25T11:00:00-07:00",
		strconv.Itoa(365*24) + "h", "2024-05-13T18:00:00Z", false, true},
}

func TestNextJobTime(t *testing.T) {
	conf := DefaultConfig()
	logger := log.Default()
	r, err := time.Parse(time.RFC3339, "2023-11-28T13:05:00Z")
	require.NoError(t, err)
	ran := &mockTimeAdaptor{time: r}
	h := New(conf, logger, ran, nil, nil, nil, nil)

	for _, tc := range nextRunTimeTestCases {
		t.Run(tc.description, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.nowS)
			require.NoError(t, err)
			offset, err := time.Parse(time.RFC3339, tc.offsetS)
			require.NoError(t, err)
			interval, err := time.ParseDuration(tc.intervalS)
			require.NoError(t, err)
			xNext, err := time.Parse(time.RFC3339, tc.xNextS)
			require.NoError(t, err)

			next, jobNow, wasAdvanced := h.next(now, offset, interval)
			assert.Equal(t, tc.xRunNow, jobNow, "jobNow")
			assert.Equal(t, xNext, next, "next")
			assert.Equal(t, tc.xWasAdvanced, wasAdvanced, "wasAdvanced")
		})
	}
}

func TestMultiError(t *testing.T) {
	var err, newErr, existErr error
	b := multiError(&err, newErr)
	assert.False(t, b)
	newErr = gerrs.New("new error")
	b = multiError(&err, newErr)
	assert.True(t, b)
	assert.Equal(t, newErr, err)
	existErr = gerrs.New("existing error")
	err = existErr
	b = multiError(&err, newErr)
	assert.True(t, b)
	assert.NotEqual(t, existErr, err)
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

	logger := log.Default()
	ran := &mockTimeAdaptor{}

	ip, err := netip.ParseAddr("0.0.0.0")
	require.NoError(t, err)
	ipService := &mockIPAdaptor{ip: ip}

	ip, err = netip.ParseAddr("1.2.3.4")
	require.NoError(t, err)
	ipCache := &mockIPAdaptor{ip: ip}

	chat := &mockChatAdaptor{c: make(chan string)}

	n, err := time.Parse(time.RFC3339Nano, "2023-11-28T00:00:00.1Z")
	require.NoError(t, err)
	now := &mockNowAdapter{now: n}

	h := New(conf, logger, ran, ipService, ipCache, chat, now)

	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		h.Start(ctx)
		wg.Done()
	}()

	time.Sleep(500 * time.Millisecond)
	now.now, err = time.Parse(time.RFC3339Nano, "2023-11-28T00:00:01Z")
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, "", chat.postChanID)
	assert.Equal(t, "0.0.0.0", chat.postMsg)

	chat.c <- "1234"

	time.Sleep(time.Second)

	assert.Equal(t, "1234", chat.postChanID)
	assert.Equal(t, "0.0.0.0", chat.postMsg)

	cancel()
	wg.Wait()
}

func TestGetIP(t *testing.T) {
	e := errors.New("An error")
	r, err := time.Parse(time.RFC3339, "2023-11-28T13:05:00Z")
	require.NoError(t, err)

	conf := DefaultConfig()
	logger := log.Default()
	ran := &mockTimeAdaptor{}
	ipService := &mockIPAdaptor{err: e}
	ipCache := &mockIPAdaptor{err: e}
	h := New(conf, logger, ran, ipService, ipCache, nil, nil)

	_, err = h.getIP(r, true)
	assert.Error(t, err)

	_, err = h.getIP(r, false)
	assert.Error(t, err)

	ipCache.err = nil
	ipCache.ip, err = netip.ParseAddr("0.0.0.0")
	require.NoError(t, err)
	ip, err := h.getIP(r, true)
	assert.NoError(t, err)
	assert.Equal(t, ipCache.ip, ip)

	ipService.err = nil
	ipService.ip, err = netip.ParseAddr("1.2.3.4")
	require.NoError(t, err)
	ip, err = h.getIP(r, false)
	assert.NoError(t, err)
	assert.Equal(t, ipService.ip, ip)
	assert.Equal(t, ipService.ip, ipCache.putIP)
	assert.Equal(t, r, ran.putTime)
}
