package hnoss

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"time"

	"github.com/nightlyone/lockfile"
)

type (
	// Hnoss is the main application object, configurable by dependency injection.
	Hnoss struct {
		config           *Config
		logger           *Logger
		ranAdapter       TimeAdapter
		ipServiceAdapter IPServiceAdapter
		ipCacheAdapter   IPAdapter
		chatAdapter      ChatAdapter
		nowAdapter       NowAdapter
		ran              time.Time
		ip               netip.Addr
	}
	// TimeAdapter should persist a time.Time
	TimeAdapter interface {
		Get() (time.Time, error)
		Put(time.Time) error
	}
	// IPAdapter should persist a netip.Addr
	IPAdapter interface {
		Get() (netip.Addr, error)
		Put(netip.Addr) error
	}
	// IPServiceAdapter should get the WAN IP address from an external service.
	IPServiceAdapter interface {
		Get() (netip.Addr, error)
	}
	// ChatAdapter should provide an interface to a chat service.
	ChatAdapter interface {
		// Chan returns a channel on which is sent a chat channelID whenever the bot should reply.
		Chan() <-chan string
		// Listen opens a chat session, if a session is already open Listen should not return an error.
		Listen() error
		Close() error
		// Post msg to the chat channel identified by chanID
		Post(chanID, msg string) error
	}
	// NowAdapter should return the current time.
	NowAdapter interface {
		Now() time.Time
	}
)

var maxTime = time.Unix(1<<63-62135596801, 999999999)
var zeroTime = time.Time{}

func New(conf *Config, logger *Logger, ranAdapter TimeAdapter, ipServiceAdapter IPServiceAdapter,
	ipCacheAdapter IPAdapter, chatAdapter ChatAdapter, nowAdapter NowAdapter) *Hnoss {
	h := &Hnoss{
		config:           conf,
		logger:           logger,
		ranAdapter:       ranAdapter,
		ipServiceAdapter: ipServiceAdapter,
		ipCacheAdapter:   ipCacheAdapter,
		chatAdapter:      chatAdapter,
		nowAdapter:       nowAdapter,
	}
	return h
}

// Start starts the scheduler.
func (h *Hnoss) Start(ctx context.Context) {
	h.logger.Log(NewInfo("scheduler started"))

	var now, next time.Time
	var runNow, wasAdvanced bool
	timer := time.NewTimer(time.Until(maxTime))
	done := ctx.Done()
	call := h.chatAdapter.Chan()

	if _, err := h.getIP(true); err != nil {
		h.logger.Log(err)
	}
	if err := h.chatAdapter.Listen(); err != nil {
		h.logger.Log(err)
	}

	for {
		now = h.nowAdapter.Now()
		next, runNow, wasAdvanced = h.next(now, h.config.Offset, h.config.Interval)

		if runNow {
			h.logger.Log(NewInfo("scheduled run missed, running now"))
			h.run(now, false, "")
		}
		stopTimer(timer)
		timer.Reset(time.Until(next))

		select {
		case <-timer.C:
			h.run(next, wasAdvanced, "")
		case chanID := <-call:
			now = time.Now().UTC()
			h.run(now, wasAdvanced, chanID)
		case <-done:
			h.logger.Log(NewInfo("exiting scheduler"))
			if err := h.chatAdapter.Close(); err != nil {
				h.logger.Log(err)
			}
			stopTimer(timer)
			return
		}
	}
}

// Stop the timer and drain the channel, if necessary.
func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		// Non-blocking because sometimes timer.Stop() can be false while timer.C is empty.
		select {
		case <-timer.C:
		default:
		}
	}
}

// Get the ip address and post it, if necessary.
func (h *Hnoss) run(t time.Time, cached bool, chanID string) {

	// Record run after.
	defer func() {
		h.ran = t
		if err := h.ranAdapter.Put(t); err != nil {
			h.logger.Log(err)
		}
	}()

	// Call Listen again each run to make sure we're connected.
	if err := h.chatAdapter.Listen(); err != nil {
		var w *Warn
		if !errors.As(err, &w) {
			h.logger.Log(err)
			var e *Error
			if errors.As(err, &e) {
				return
			}
		}
	}

	cur := h.ip
	ip, err := h.getIP(cached)
	if err != nil {
		h.logger.Log(err)
		return
	}

	post := false
	if chanID != "" {
		h.logger.Log(Infof("replying to message on channel %s", chanID))
		post = true
	}
	if cur != ip {
		h.logger.Log(Infof("ip address changed from %s to %s", cur.String(), ip.String()))
		post = true
	}
	if post {
		if err = h.chatAdapter.Post(chanID, fmt.Sprintf(h.config.IPMessageFormat, ip.String())); err != nil {
			h.logger.Log(err)
		}
		return
	}
	h.logger.Log(NewInfo("ip address unchanged"))
}

// Get the next run time.
func (h *Hnoss) next(now, offset time.Time, interval time.Duration) (
	next time.Time, runNow, wasAdvanced bool) {

	dif := time.Duration(offset.UnixNano())%interval - time.Duration(now.UnixNano())%interval
	if dif == 0 {
		dif += interval
		runNow = true
	} else if dif < 0 {
		dif += interval
	}
	next = now.Add(dif)
	expected := next.Add(-interval)

	// If ran can't be found, or if ran is before expected, run now.
	prev, err := h.getRan()
	if err != nil {
		h.logger.Log(err)
		runNow = true
		return
	}
	if prev.Before(expected) {
		runNow = true
		return
	}

	// Scheduled time brought forward
	// e.g. scheduled time 11:05, ran time was 10:30, so next time is 12:05
	if prev.After(expected) {
		next = next.Add(interval)
		wasAdvanced = true
	}

	return
}

func (h *Hnoss) getRan() (time.Time, error) {
	var err error
	if h.ran.Equal(zeroTime) {
		h.ran, err = h.ranAdapter.Get()
		if err != nil {
			return zeroTime, err
		}
	}
	return h.ran, nil
}

func (h *Hnoss) getIP(cached bool) (netip.Addr, error) {
	if !cached {
		ip, err := h.ipServiceAdapter.Get()
		if err != nil {
			return h.ip, err
		}
		h.ip = ip
		if err = h.ipCacheAdapter.Put(ip); err != nil {
			h.logger.Log(err)
		}
	} else if !h.ip.IsValid() {
		var err error
		h.ip, err = h.ipCacheAdapter.Get()
		if err != nil {
			return netip.Addr{}, err
		}
	}
	return h.ip, nil
}

func Lock(pidFile string) (func() error, error) {
	p, err := filepath.Abs(pidFile)
	if err != nil {
		return nil, FatalWrapf(err, "failed to get absolute path for PIDFile: %s", pidFile)
	}
	err = mkDir(p, "lock")
	if err != nil {
		return nil, (*Fatal)(err.(*Error))
	}
	lock, err := lockfile.New(p)
	if err != nil {
		return nil, FatalWrap(err, "failed to init lock")
	}
	if err = lock.TryLock(); err != nil {
		return nil, FatalWrap(err, "failed to lock")
	}
	return func() error {
		if err := lock.Unlock(); err != nil {
			return FatalWrap(err, "failed to unlock")
		}
		return nil
	}, nil
}

func PanicOnError(f func() error) {
	if f == nil {
		return
	}
	if err := f(); err != nil {
		panic(err)
	}
}
