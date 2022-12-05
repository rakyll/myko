package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	pb "github.com/mykodev/myko/proto"
)

var (
	op     string
	n      int
	events int
)

func main() {
	ctx := context.Background()

	flag.StringVar(&op, "op", "insert", "")
	flag.IntVar(&n, "n", 1000, "")
	flag.IntVar(&events, "events", 10, "")
	flag.Parse()

	client := pb.NewServiceJSONClient("http://localhost:6959", &http.Client{})
	switch op {
	case "insert":
		benchmarkInserts(ctx, client)
	default:
		log.Fatalf("unknown benchmark op")
	}
}

func benchmarkInserts(ctx context.Context, client pb.Service) {
	var entries []*pb.Entry
	for i := 0; i < events; i++ {
		entries = append(entries, &pb.Entry{
			TraceId: "",
			Origin:  "test_origin",
			Events: []*pb.Event{
				{
					Name:  "event_1",
					Unit:  "unit",
					Value: 12.5,
				},
			},
		})
	}

	for i := 0; i < n; i++ {
		start := time.Now()
		_, err := client.InsertEvents(ctx, &pb.InsertEventsRequest{
			Entries: entries,
		})
		if err == nil {
			log.Printf("Insert responded in %v", time.Now().Sub(start))
		} else {
			log.Printf("Insert errored with %v", err)
		}
	}
}
