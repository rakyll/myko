package server

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/aggregator"
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

	summer := aggregator.NewSum(128)
	for q.Iter().Scan(&name, &value, &unit) {
		summer.Add(req.TraceId, req.Origin, &pb.Event{
			Name:  name,
			Value: value,
			Unit:  unit,
		})
	}

	events := make([]*pb.Event, summer.Size())
	summer.ForEach(func(traceID, origin string, event *pb.Event) error {
		events = append(events, event)
		return nil
	})

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
		summer:        aggregator.NewSum(n),
	}
}

type batchWriter struct {
	mu     sync.Mutex // guards summer
	summer *aggregator.Summer

	n             int
	flushInterval time.Duration
	lastExport    time.Time

	server *Server
}

func (b *batchWriter) Write(e *pb.Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ev := range e.Events {
		b.summer.Add(e.TraceId, e.Origin, ev)

	}
	return b.flushIfNeeded()
}

func (b *batchWriter) flushIfNeeded() error {
	// flushIfNeeded need to be called from Write.
	if size := b.summer.Size(); size > b.n || b.lastExport.Before(time.Now().Add(-1*b.flushInterval)) {
		log.Printf("Batch writing %d records", size)

		batch := b.server.session.NewBatch(gocql.UnloggedBatch)
		if err := b.summer.ForEach(func(traceID, origin string, ev *pb.Event) error {
			id, err := gocql.RandomUUID()
			if err != nil {
				return err
			}
			return batch.Query(`
				INSERT INTO {{.Keyspace}}.events
				(id, trace_id, origin, event, value, unit, created_at)
				VALUES ( ?, ?, ?, ?, ?, ?, ? )
				USING TTL {{.TTL}}`,
				id.String(), traceID, origin, ev.Name, ev.Value, ev.Unit, time.Now())
		}); err != nil {
			return err
		}
		if err := b.server.session.ExecuteBatch(batch); err != nil {
			// TODO: Retry and drop the samples if retries fail.
			return err
		}
		b.summer.Reset()
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
