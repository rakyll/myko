package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string `yaml:"listen"`

	DataConfig DataConfig `yaml:"data"`

	FlushConfig FlushConfig `yaml:"flush"`
}

func DefaultConfig() Config {
	return Config{
		Listen: ":6959",
		DataConfig: DataConfig{
			TTL: 24 * time.Hour,
			CassandraConfig: CassandraConfig{
				Keyspace: "myko",
				Peers:    []string{"localhost:9042"},
				Timeout:  30 * time.Second,
			},
		},
		FlushConfig: FlushConfig{
			BufferSize: 8 * 1024,
			Interval:   60 * time.Second,
		},
	}
}

type DataConfig struct {
	TTL time.Duration `yaml:"ttl"`

	CassandraConfig CassandraConfig `yaml:"cassandra"`

	KustoConfig KustoConfig `yaml:"kusto"`
}

type KustoConfig struct {
	Endpoint string `yaml:"endpoint,omitempty"`

	ClientID string `yaml:"client_id,omitempty"`

	ClientSecret string `yaml:"client_secret,omitempty"`

	TenantID string `yaml:"tenant_id,omitempty"`

	Database string `yaml:"database,omitempty"`

	Table string `yaml:"table,omitempty"`
}

type CassandraConfig struct {
	Keyspace string `yaml:"keyspace,omitempty"`

	Peers []string `yaml:"peers,omitempty"`

	Username string `yaml:"username,omitempty"`

	Password string `yaml:"password,omitempty"`

	Datacenter string `yaml:"dc,omitempty"`

	Timeout time.Duration `yaml:"timeout,omitempty"`

	SSLSkipVerify bool `yaml:"ssl_skip_verify,omitempty"`
}

type FlushConfig struct {
	// BufferSize is the uppermost size of the data points
	// kept in-memory before they are flushed out to the datastore.
	BufferSize int `yaml:"buffer_size"`

	// Interval is the uppermost duration to wait before
	// all in-memory data points are flushed out to the datastore.
	Interval time.Duration `yaml:"interval"`
}

func Open(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	config := DefaultConfig()
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return Config{}, err
	}
	return config, nil
}
