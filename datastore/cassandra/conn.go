package cassandra

import (
	"errors"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/config"
)

func NewSession(c *config.CassandraConfig) (sess *gocql.Session, err error) {
	if len(c.Peers) == 0 {
		return nil, errors.New("no peers given")
	}
	// TODO: Partition by date?
	cluster := gocql.NewCluster(c.Peers...)
	cluster.Timeout = c.Timeout
	if c.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: c.Username, Password: c.Password}
	}
	if c.Datacenter != "" {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(c.Datacenter)
	}

	if len(c.Peers) == 1 {
		cluster.Consistency = gocql.LocalOne
	} else {
		cluster.Consistency = gocql.Quorum
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
	`CREATE INDEX IF NOT EXISTS createdAtIndex ON events.data ( created_at );`,
}
