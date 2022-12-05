package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/mykodev/myko/datastore/scylladb"
	pb "github.com/mykodev/myko/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	listen           string
	database         string // comma separated list of peers
	databaseUser     string
	databasePassword string
	databaseDC       string
	timeout          time.Duration
)

func main() {
	flag.StringVar(&listen, "listen", ":6959", "")
	flag.StringVar(&database, "scylladb", "localhost:9043", "")
	flag.StringVar(&databaseUser, "scylladb-user", "", "")
	flag.StringVar(&databasePassword, "scylladb-passwd", "", "")
	flag.StringVar(&databaseDC, "scylladb-dc", "", "")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "")
	flag.Parse()

	session, err := scylladb.NewSession(scylladb.Options{
		Peers:          strings.Split(database, ","),
		User:           databaseUser,
		Password:       databasePassword,
		Datacenter:     databaseDC,
		DefaultTimeout: timeout,
	})
	if err != nil {
		log.Fatalf("Failed to create a connection to datastore: %v", err)
	}

	log.Printf("Starting the myko server at %q...", listen)
	server := pb.NewServiceServer(&service{session: session}, nil)
	log.Fatal(http.ListenAndServe(listen, server))
}

type service struct {
	session *gocql.Session
}

func (s *service) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsResponse, error) {
	filter := scylladb.Filter{
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
	batch := s.session.NewBatch(gocql.UnloggedBatch)
	for _, entry := range req.Entries {
		for _, e := range entry.Events {
			id, err := gocql.RandomUUID()
			if err != nil {
				return nil, err
			}
			if !e.CreatedAt.IsValid() {
				e.CreatedAt = timestamppb.Now()
			}
			batch.Query(`
			INSERT INTO events.data 
			(id, trace_id, origin, event, value, unit, created_at)
			VALUES ( ?, ?, ?, ?, ?, ?, ? )`,
				id.String(), entry.TraceId, entry.Origin, e.Name, e.Value, e.Unit, e.CreatedAt.AsTime())
		}
	}
	if err := s.session.ExecuteBatch(batch); err != nil {
		return nil, err
	}
	return &pb.InsertEventsResponse{}, nil
}

func (s *service) DeleteEvents(ctx context.Context, req *pb.DeleteEventsRequest) (*pb.DeleteEventsResponse, error) {
	filter := scylladb.Filter{
		TraceID: req.TraceId,
		Origin:  req.Origin,
		Event:   req.Event,
	}
	filterCQL, err := filter.CQL()
	if err != nil {
		return nil, err
	}

	if err := s.session.Query(`
		DELETE
		FROM events.data
		` + filterCQL + ` ALLOW FILTERING`).Exec(); err != nil {
		return nil, err
	}
	return &pb.DeleteEventsResponse{}, nil
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
