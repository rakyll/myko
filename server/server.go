package server

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/config"
	"github.com/mykodev/myko/datastore/cassandra"
	"github.com/mykodev/myko/format"

	pb "github.com/mykodev/myko/proto"
)

type Server struct {
	session     *cassandra.Session
	batchWriter *batchWriter
}

func New(cfg config.Config) (*Server, error) {
	session, err := cassandra.NewSession(cfg.DataConfig)
	if err != nil {
		return nil, err
	}
	server := &Server{session: session}
	server.batchWriter = newBatchWriter(server, cfg.FlushConfig.BufferSize, cfg.FlushConfig.Interval)
	return server, nil
}

func (s *Server) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	filter := cassandra.Filter{
		TraceID: req.TraceId,
		Origin:  req.Origin,
		Event:   req.Event,
	}
	filterCQL, err := filter.CQL()
	if err != nil {
		return nil, err
	}

	q, err := s.session.Query(`
		SELECT event, value, unit 
		FROM {{.Keyspace}}.events ` + filterCQL + ` ALLOW FILTERING`)
	if err != nil {
		return nil, err
	}

	var (
		name  string
		unit  string
		value float64
	)

	v := make(map[string]*pb.Event)
	for q.Iter().Scan(&name, &value, &unit) {
		k := key(req.TraceId, req.Origin, name, unit)
		event, ok := v[k]
		if ok {
			event.Value += value
			v[k] = event
		} else {
			v[k] = &pb.Event{
				Name:  name,
				Value: value,
				Unit:  unit,
			}
		}
	}
	var events []*pb.Event
	for _, e := range v {
		events = append(events, &pb.Event{
			Name:  e.Name,
			Unit:  e.Unit,
			Value: e.Value,
		})
	}

	sEvents := sortableEvents(events)
	sort.Sort(sEvents)
	return &pb.QueryResponse{Events: sEvents}, nil
}

func (s *Server) InsertEvents(ctx context.Context, req *pb.InsertEventsRequest) (*pb.InsertEventsResponse, error) {
	for _, entry := range req.Entries {
		if err := s.batchWriter.Write(format.Espace(entry)); err != nil {
			return nil, err
		}
	}
	return &pb.InsertEventsResponse{}, nil
}

func (s *Server) DeleteEvents(ctx context.Context, req *pb.DeleteEventsRequest) (*pb.DeleteEventsResponse, error) {
	filter := cassandra.Filter{
		TraceID: req.TraceId,
		Origin:  req.Origin,
		Event:   req.Event,
	}
	filterCQL, err := filter.CQL()
	if err != nil {
		return nil, err
	}

	q, err := s.session.Query(`SELECT id FROM {{.Keyspace}}.events ` + filterCQL + ` ALLOW FILTERING`)
	if err != nil {
		return nil, err
	}

	var id gocql.UUID
	for q.Iter().Scan(&id) {
		// TODO: Replace deletion with TTL on events table.
		log.Printf("Deleting %q", id)

		q, err := s.session.Query(`DELETE FROM {{.Keyspace}}.events WHERE id = ?`, id)
		if err != nil {
			return nil, err
		}
		if err := q.Exec(); err != nil {
			return nil, err
		}
	}
	return &pb.DeleteEventsResponse{}, nil
}

func newBatchWriter(server *Server, n int, flushInterval time.Duration) *batchWriter {
	// TODO: Implement an optional WAL.
	return &batchWriter{
		server:        server,
		n:             n,
		flushInterval: flushInterval,
		events:        make(map[string]*pb.Event, n),
	}
}

type batchWriter struct {
	mu     sync.Mutex // guards events
	events map[string]*pb.Event

	n             int
	flushInterval time.Duration
	lastExport    time.Time

	server *Server
}

func (b *batchWriter) Write(e *pb.Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, event := range e.Events {
		key := key(e.Origin, e.TraceId, event.Name, event.Unit)
		v, ok := b.events[key]
		if !ok {
			b.events[key] = event
		} else {
			v.Value += event.Value
			b.events[key] = v
		}
	}
	return b.flushIfNeeded()
}

func (b *batchWriter) flushIfNeeded() error {
	// flushIfNeeded need to be called from Write.
	if len(b.events) > b.n || b.lastExport.Before(time.Now().Add(-1*b.flushInterval)) {
		log.Printf("Batch writing %d records", len(b.events))

		batch := b.server.session.NewBatch(gocql.UnloggedBatch)
		for key, e := range b.events {
			origin, traceID, name, unit := parseKey(key)

			id, err := gocql.RandomUUID()
			if err != nil {
				return err
			}
			if err := batch.Query(`
				INSERT INTO {{.Keyspace}}.events
				(id, trace_id, origin, event, value, unit, created_at)
				VALUES ( ?, ?, ?, ?, ?, ?, ? )
				USING TTL {{.TTL}}`,
				id.String(), traceID, origin, name, e.Value, unit, time.Now()); err != nil {
				return err
			}
		}
		if err := b.server.session.ExecuteBatch(batch); err != nil {
			// TODO: Retry and drop the samples if retries fail.
			return err
		}
		b.events = make(map[string]*pb.Event, b.n)
		b.lastExport = time.Now()
	}
	return nil
}

type sortableEvents []*pb.Event

func (s sortableEvents) Len() int {
	return len(s)
}

func (s sortableEvents) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s sortableEvents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func key(origin, traceID, name, unit string) string {
	return origin + ":" + traceID + ":" + name + ":" + unit
}

func parseKey(key string) (origin, traceID, name, unit string) {
	v := strings.Split(key, ":")
	return v[0], v[1], v[2], v[3]
}
