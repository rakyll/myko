package cassandra

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/config"
)

type Session struct {
	ttl      int64
	keyspace string
	session  *gocql.Session
}

func NewSession(dataConfig config.DataConfig) (*Session, error) {
	c := dataConfig.CassandraConfig

	if len(c.Peers) == 0 {
		return nil, errors.New("no peers given")
	}
	clusterConf := gocql.NewCluster(c.Peers...)
	clusterConf.Timeout = c.Timeout
	if c.Username != "" {
		clusterConf.Authenticator = gocql.PasswordAuthenticator{Username: c.Username, Password: c.Password}
	}
	if c.Datacenter != "" {
		clusterConf.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(c.Datacenter)
	}
	clusterConf.ProtoVersion = 4

	if len(c.Peers) == 1 {
		clusterConf.Consistency = gocql.LocalOne
	} else {
		clusterConf.Consistency = gocql.Quorum
	}

	session, err := clusterConf.CreateSession()
	if err != nil {
		return nil, err
	}

	s := &Session{
		ttl:      int64(dataConfig.TTL) / (1000 * 1000), // in seconds
		keyspace: c.Keyspace,
		session:  session,
	}
	for _, q := range initCQLs {
		query, err := s.Query(q)
		if err != nil {
			return nil, fmt.Errorf("failed create query for %q: %v", q, err)
		}
		if err = query.Exec(); err != nil {
			return nil, fmt.Errorf("failed to run %q: %v", q, err)
		}
	}
	return s, nil
}

func (s *Session) Query(q string, vals ...interface{}) (*gocql.Query, error) {
	tmpl, err := template.New(q).Parse(q)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &queryData{
		Keyspace: s.keyspace,
		TTL:      s.ttl,
	}); err != nil {
		return nil, err
	}
	return s.session.Query(buf.String(), vals...), nil
}

func (s *Session) NewBatch(bt gocql.BatchType) *Batch {
	return &Batch{
		session: s,
		batch:   gocql.NewBatch(bt),
	}
}

type Batch struct {
	session *Session
	batch   *gocql.Batch
}

func (b *Batch) Query(q string, vals ...interface{}) error {
	tmpl, err := template.New(q).Parse(q)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &queryData{
		Keyspace: b.session.keyspace,
		TTL:      b.session.ttl,
	}); err != nil {
		return err
	}
	b.batch.Query(buf.String(), vals...)
	return nil
}

func (s *Session) ExecuteBatch(b *Batch) error {
	return s.session.ExecuteBatch(b.batch)
}

type queryData struct {
	Keyspace string
	TTL      int64
}

var initCQLs = []string{
	// TODO: Choose a migration tool before the release.
	`CREATE KEYSPACE IF NOT EXISTS {{.Keyspace}}
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}
		AND durable_writes = true;`,
	`CREATE TABLE IF NOT EXISTS {{.Keyspace}}.events (
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
	`CREATE INDEX IF NOT EXISTS traceIndex ON {{.Keyspace}}.events ( trace_id );`,
	`CREATE INDEX IF NOT EXISTS originIndex ON {{.Keyspace}}.events ( origin );`,
	`CREATE INDEX IF NOT EXISTS eventIndex ON {{.Keyspace}}.events ( event );`,
	`CREATE INDEX IF NOT EXISTS createdAtIndex ON {{.Keyspace}}.events ( created_at );`,
}
