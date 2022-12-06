package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string `yaml:"listen"`

	DataConfig *DataConfig `yaml:"data"`

	FlushConfig *FlushConfig `yaml:"flush"`
}

func DefaultConfig() *Config {
	return &Config{
		Listen:      ":6959",
		DataConfig:  DefaultDataConfig(),
		FlushConfig: DefaultFlushConfig(),
	}
}

type DataConfig struct {
	CassandraConfig *CassandraConfig `yaml:"cassandra"`
}

func DefaultDataConfig() *DataConfig {
	return &DataConfig{
		CassandraConfig: &CassandraConfig{
			Peers: []string{"localhost:9042"},
		},
	}
}

type CassandraConfig struct {
	Peers []string `yaml:"peers,omitempty"`

	Username string `yaml:"username,omitempty"`

	Password string `yaml:"password,omitempty"`

	Datacenter string `yaml:"dc,omitempty"`

	Timeout time.Duration `yaml:"timeout,omitempty"`
}

type FlushConfig struct {
	Interval time.Duration `yaml:"interval,omitempty"`
}

func DefaultFlushConfig() *FlushConfig {
	return &FlushConfig{
		Interval: 5 * time.Second,
	}
}

func Open(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	defaultConfig := DefaultConfig()

	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	if config.Listen == "" {
		config.Listen = defaultConfig.Listen
	}
	if config.DataConfig == nil {
		config.DataConfig = defaultConfig.DataConfig
	}
	if config.FlushConfig == nil {
		config.FlushConfig = defaultConfig.FlushConfig
	}
	return &config, nil
}
