package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/datastore/cassandra"
	pb "github.com/mykodev/myko/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	listen           string
	database         string // comma separated list of peers
	databaseUser     string
	databasePassword string
	datacenter       string
	timeout          time.Duration
)

func main() {
	flag.StringVar(&listen, "listen", ":6959", "")
	flag.StringVar(&database, "cassandra", "localhost:9043", "")
	flag.StringVar(&databaseUser, "cassandra-user", "", "")
	flag.StringVar(&databasePassword, "cassandra-passwd", "", "")
	flag.StringVar(&datacenter, "datacenter", "", "")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "")
	flag.Parse()

	session, err := cassandra.NewSession(cassandra.Options{
		Peers:          strings.Split(database, ","),
		User:           databaseUser,
		Password:       databasePassword,
		Datacenter:     datacenter,
		DefaultTimeout: timeout,
	})
	if err != nil {
		log.Fatalf("Failed to create a connection to datastore: %v", err)
	}

	log.Printf("Starting the myko server at %q...", listen)
	server := pb.NewServiceServer(
		&service{
			session:     session,
			batchWriter: newBatchWriter(session, 100),
		}, nil)
	log.Fatal(http.ListenAndServe(listen, server))
}

type service struct {
	session     *gocql.Session
	batchWriter *batchWriter
}

func (s *service) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
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
	return &pb.ListEventsResponse{Events: sorter.events}, nil
}

func (s *service) InsertEvents(ctx context.Context, req *pb.InsertEventsRequest) (*pb.InsertEventsResponse, error) {
	for _, entry := range req.Entries {
		if err := s.batchWriter.Write(entry); err != nil {
			return nil, err
		}
	}
	return &pb.InsertEventsResponse{}, nil
}

func (s *service) DeleteEvents(ctx context.Context, req *pb.DeleteEventsRequest) (*pb.DeleteEventsResponse, error) {
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

func newBatchWriter(session *gocql.Session, n int) *batchWriter {
	return &batchWriter{
		n:       n,
		events:  make(map[string]*pb.Event, n),
		session: session,
	}
}

type batchWriter struct {
	mu      sync.Mutex
	n       int
	events  map[string]*pb.Event // by origin and trace_id
	session *gocql.Session
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

	if len(b.events) > b.n {
		log.Printf("Batch writing %d records", len(b.events))
		batch := b.session.NewBatch(gocql.UnloggedBatch)
		for key, e := range b.events {
			origin, traceID, name, unit := b.parseKey(key)

			id, err := gocql.RandomUUID()
			if err != nil {
				return err
			}
			if !e.CreatedAt.IsValid() {
				e.CreatedAt = timestamppb.Now()
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
