package scylladb

import (
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type Options struct {
	Peers      []string
	User       string
	Password   string
	Datacenter string

	DefaultTimeout time.Duration
}

func NewSession(o Options) (sess *gocql.Session, err error) {
	if len(o.Peers) == 0 {
		return nil, errors.New("no peers given")
	}
	// TODO: Partition by date?
	cluster := gocql.NewCluster(o.Peers...)
	cluster.Timeout = o.DefaultTimeout
	if o.User != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: o.User, Password: o.Password}
	}
	if o.Datacenter != "" {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(o.Datacenter)
	}

	sess, err = cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	for _, q := range initCQLs {
		if err = sess.Query(q).Exec(); err != nil {
			return nil, fmt.Errorf("Failed to run %q: %v", q, err)
		}
	}
	return sess, nil
}

var initCQLs = []string{
	// TODO: Choose a migration tool before the release.
	`CREATE KEYSPACE IF NOT EXISTS events
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}
		AND durable_writes = true;`,
	`CREATE TABLE IF NOT EXISTS events.data (
		id uuid primary key, 
		trace_id text,
		origin text,
		attr_key text,
		attr_value text,
		event text,
		unit text, 
		value double,
		created_at timestamp
	);`,
	`CREATE INDEX IF NOT EXISTS traceIndex ON events.data ( trace_id );`,
	`CREATE INDEX IF NOT EXISTS originIndex ON events.data ( origin );`,
	`CREATE INDEX IF NOT EXISTS eventIndex ON events.data ( event );`,
}
