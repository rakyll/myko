package server

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/mykodev/myko/aggregator"
	"github.com/mykodev/myko/config"
	"github.com/mykodev/myko/datastore/kusto"
	"github.com/mykodev/myko/format"

	pb "github.com/mykodev/myko/proto"
)

type Server struct {
	session     *kusto.Session
	batchWriter *batchWriter
}

func New(cfg config.Config) (*Server, error) {
	session, err := kusto.NewSession(cfg.DataConfig)
	if err != nil {
		return nil, err
	}
	server := &Server{session: session}
	server.batchWriter = newBatchWriter(server, cfg.FlushConfig.BufferSize, cfg.FlushConfig.Interval)
	return server, nil
}

func (s *Server) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	return &pb.QueryResponse{}, errors.New("not yet supported")
}

func (s *Server) InsertEvents(ctx context.Context, req *pb.InsertEventsRequest) (*pb.InsertEventsResponse, error) {
	for _, entry := range req.Entries {
		if err := format.Verify(entry); err != nil {
			return nil, err
		}
	}
	for _, entry := range req.Entries {
		if err := s.batchWriter.Write(entry); err != nil {
			return nil, err
		}
	}
	return &pb.InsertEventsResponse{}, nil
}

func newBatchWriter(server *Server, bufferSize int, flushInterval time.Duration) *batchWriter {
	return &batchWriter{
		server:        server,
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		summer:        aggregator.NewSummer(bufferSize),
	}
}

type batchWriter struct {
	mu     sync.Mutex // guards summer
	summer *aggregator.Summer

	bufferSize    int
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
	ctx := context.Background()

	// flushIfNeeded need to be called from Write.
	if size := b.summer.Size(); size >= b.bufferSize || b.lastExport.Before(time.Now().Add(-1*b.flushInterval)) {
		log.Printf("Writing %d events", size)

		if err := b.summer.ForEach(func(traceID, origin string, ev *pb.Event) error {
			return b.server.session.Ingest(ctx, traceID, origin, ev)
		}); err != nil {
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
