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
	session     *gocql.Session
	batchWriter *batchWriter
}

func New(cfg *config.Config) (*Server, error) {
	cassandraConfig := cfg.DataConfig.CassandraConfig
	if cassandraConfig == nil {
		// TODO: Allow other data stores in the future.
		log.Fatalf("No cassandra config provided")
	}
	session, err := cassandra.NewSession(cassandraConfig)
	if err != nil {
		return nil, err
	}
	return &Server{
		session:     session,
		batchWriter: newBatchWriter(session, 100, cfg.FlushConfig.Interval),
	}, nil
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

	iter := s.session.Query(`
		SELECT origin, event, value, unit
		FROM events.data
		` + filterCQL + ` ALLOW FILTERING`).Iter()

	var (
		origin string
		name   string
		unit   string
		value  float64
	)

	key := func(origin, event, unit string) string {
		return origin + ":" + event + ":" + unit
	}

	v := make(map[string]*pb.Event)
	for iter.Scan(&origin, &name, &value, &unit) {
		k := key(origin, name, unit)
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

	sorter := &eventSorter{events: events}
	sort.Sort(sorter)
	return &pb.QueryResponse{Events: sorter.events}, nil
}

func (s *Server) InsertEvents(ctx context.Context, req *pb.InsertEventsRequest) (*pb.InsertEventsResponse, error) {
	for _, entry := range req.Entries {
		entry = format.Espace(entry)
		if err := s.batchWriter.Write(entry); err != nil {
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

	iter := s.session.Query(`SELECT id FROM events.data ` +
		filterCQL + ` ALLOW FILTERING`).Iter()
	var id gocql.UUID
	for iter.Scan(&id) {
		log.Printf("Deleting %q", id)
		if err := s.session.Query(`DELETE FROM events.data WHERE id = ?`, id.String()).Exec(); err != nil {
			return nil, err
		}
	}
	return &pb.DeleteEventsResponse{}, nil
}

func newBatchWriter(session *gocql.Session, n int, flushInterval time.Duration) *batchWriter {
	// TODO: Implement an optional WAL.
	return &batchWriter{
		session:       session,
		n:             n,
		flushInterval: flushInterval,
		events:        make(map[string]*pb.Event, n),
	}
}

type batchWriter struct {
	mu         sync.Mutex
	events     map[string]*pb.Event
	lastExport time.Time

	n             int
	flushInterval time.Duration
	session       *gocql.Session
}

func (b *batchWriter) key(origin, traceID, name, unit string) string {
	return origin + ":" + traceID + ":" + name + ":" + unit
}

func (b *batchWriter) parseKey(key string) (origin, traceID, name, unit string) {
	v := strings.Split(key, ":")
	return v[0], v[1], v[2], v[3]
}

func (b *batchWriter) Write(e *pb.Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, event := range e.Events {
		key := b.key(e.Origin, e.TraceId, event.Name, event.Unit)
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
		batch := b.session.NewBatch(gocql.LoggedBatch)
		for key, e := range b.events {
			origin, traceID, name, unit := b.parseKey(key)

			id, err := gocql.RandomUUID()
			if err != nil {
				return err
			}
			batch.Query(`
				INSERT INTO events.data 
				(id, trace_id, origin, event, value, unit, created_at)
				VALUES ( ?, ?, ?, ?, ?, ?, ? )`,
				id.String(), origin, traceID, name, e.Value, unit, time.Now())
		}
		if err := b.session.ExecuteBatch(batch); err != nil {
			return err
		}
		b.events = make(map[string]*pb.Event, b.n)
		b.lastExport = time.Now()
	}
	return nil
}

type eventSorter struct {
	events []*pb.Event
}

func (s *eventSorter) Len() int {
	return len(s.events)
}

func (s *eventSorter) Less(i, j int) bool {
	return s.events[i].Name < s.events[j].Name
}

func (s *eventSorter) Swap(i, j int) {
	s.events[i], s.events[j] = s.events[j], s.events[i]
}
