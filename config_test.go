package hnoss

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	conf, err := ConfigureFromFile("testdata/hnoss.yaml")
	require.NoError(t, err)

	offset, err := time.Parse(time.RFC3339, "1977-05-25T11:00:00-07:00")
	require.NoError(t, err)

	expected := &Config{
		Interval:                  time.Hour * 2,
		Offset:                    offset,
		PIDFile:                   "run/pid",
		RanFile:                   "run/ran",
		IPServiceURL:              "http://localhost:45782/ip",
		IPCacheFile:               "run/ip",
		IPMessageFormat:           "%s:2456",
		DiscordBotToken:           "1234",
		DiscordDefaultChannelName: "valheim",
	}

	assert.Equal(t, expected, conf)
}
