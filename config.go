package hnoss

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		Interval                  time.Duration
		Offset                    time.Time
		PIDFile                   string
		RanFile                   string
		IPServiceURL              string
		IPCacheFile               string
		IPMessageFormat           string
		DiscordBotToken           string
		DiscordDefaultChannelName string
		LogFile                   string
	}
	yamlConfig struct {
		Interval                  string `yaml:"interval"`
		Offset                    string `yaml:"offset"`
		PIDFile                   string `yaml:"pidFile"`
		RanFile                   string `yaml:"ranFile"`
		IPServiceURL              string `yaml:"ipServiceURL"`
		IPCacheFile               string `yaml:"ipCacheFile"`
		IPMessageFormat           string `yaml:"ipMessageFormat"`
		DiscordBotToken           string `yaml:"discordBotToken"`
		DiscordDefaultChannelName string `yaml:"discordDefaultChannelName"`
		LogFile                   string `yaml:"logFile"`
	}
)

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	y := defaultYAMLConfig()
	var err error
	if err = value.Decode(&y); err != nil {
		return err
	}
	return c.Set(y)
}

func (c *Config) Set(y *yamlConfig) error {
	var err error
	c.Interval, err = time.ParseDuration(y.Interval)
	if err != nil {
		return ErrorWrapf(err, "config: failed to parse interval: %s", y.Interval)
	}
	c.Offset, err = time.Parse(time.RFC3339, y.Offset)
	if err != nil {
		return ErrorWrapf(err, "config: failed to parse offset: %s", y.Offset)
	}

	c.PIDFile = y.PIDFile
	c.RanFile = y.RanFile
	c.IPServiceURL = y.IPServiceURL
	c.IPCacheFile = y.IPCacheFile
	c.IPMessageFormat = y.IPMessageFormat
	c.DiscordBotToken = y.DiscordBotToken
	c.DiscordDefaultChannelName = y.DiscordDefaultChannelName
	c.LogFile = y.LogFile
	return nil
}

func DefaultConfig() *Config {
	y := defaultYAMLConfig()
	c := &Config{}
	err := c.Set(y)
	if err != nil {
		panic(err)
	}
	return c
}

func defaultYAMLConfig() *yamlConfig {
	return &yamlConfig{
		Interval:        "1h",
		Offset:          "2023-11-28T00:00:00Z",
		PIDFile:         "/run/hnoss.pid",
		RanFile:         "/var/cache/hnoss/ran",
		IPCacheFile:     "/var/cache/hnoss/ip",
		IPMessageFormat: "%s",
		LogFile:         "/var/log/hnoss.log",
	}
}

func ConfigureFromFile(path string) (conf *Config, err error) {
	file, closeFile := openFile(path, "config", &err)
	if err != nil {
		err = (*Fatal)(err.(*Error))
		return
	}
	defer closeFile()
	conf, err = ConfigureFromReadSeeker(file)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func ConfigureFromReadSeeker(reader io.ReadSeeker) (*Config, error) {
	_, err := reader.Seek(0, 0)
	if err != nil {
		return nil, FatalWrap(err, "failed to rewind config file")
	}
	decoder := yaml.NewDecoder(reader)
	conf := &Config{}
	err = decoder.Decode(conf)
	if err != nil {
		return nil, FatalWrap(err, "failed to decode config file")
	}

	return conf, nil
}

func openFile(path, desc string, err *error) (file *os.File, closeFile func()) {
	file, *err = os.Open(path)
	if *err != nil {
		*err = ErrorWrapf(*err, "failed to open %s file: %s", desc, path)
	}
	closeFile = closeFileFunc(path, desc, err, file)
	return
}

func createFile(path, desc string, err *error) (file *os.File, closeFile func()) {
	*err = mkDir(path, desc)
	file, *err = os.Create(path)
	if *err != nil {
		*err = ErrorWrapf(*err, "failed to create %s file: %s", desc, path)
	}
	closeFile = closeFileFunc(path, desc, err, file)
	return
}

func mkDir(path, desc string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return ErrorWrapf(err, "failed to make directory for %s file: %s", desc, path)
	}
	return nil
}

func closeFileFunc(path, desc string, err *error, file *os.File) func() {
	return func() {
		if fErr := file.Close(); fErr != nil {
			fErr = ErrorWrapf(fErr, "failed to close %s file: %s", desc, path)
			multiError(err, fErr)
		}
	}
}
