package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/montanaflynn/stats"
	pb "github.com/mykodev/myko/proto"
)

var (
	op                 string
	n                  int
	events             int
	traceIDCardinality int
	eventCardinality   int
	unitCardinality    int

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const benchmarkOrigin = "__reserved_myko_benchmark_origin"

func main() {
	ctx := context.Background()

	flag.StringVar(&op, "op", "insert", "")
	flag.IntVar(&n, "n", 1000, "")
	flag.IntVar(&events, "events", 10, "")
	flag.IntVar(&traceIDCardinality, "trace-id-cardinality", 100, "")
	flag.IntVar(&eventCardinality, "event-cardinality", 20, "")
	flag.IntVar(&unitCardinality, "unit-cardinality", 10, "")
	flag.Parse()

	client := pb.NewServiceJSONClient("http://localhost:6959", &http.Client{})
	switch op {
	case "insert":
		benchmarkInserts(ctx, client)
	case "cleanup":
		cleanup(ctx, client)
	default:
		log.Fatalf("unknown benchmark op")
	}
}

func benchmarkInserts(ctx context.Context, client pb.Service) {
	var entries []*pb.Entry
	for i := 0; i < events; i++ {
		entries = append(entries, &pb.Entry{
			TraceId: fmt.Sprintf("trace_%d", random.Intn(traceIDCardinality)),
			Origin:  benchmarkOrigin,
			Events: []*pb.Event{
				{
					Name:  fmt.Sprintf("event_%d", random.Intn(eventCardinality)),
					Unit:  fmt.Sprintf("unit_%d", random.Intn(unitCardinality)),
					Value: random.Float64(),
				},
			},
		})
	}

	s := newSummary(n)
	for i := 0; i < n; i++ {
		start := time.Now()
		_, err := client.InsertEvents(ctx, &pb.InsertEventsRequest{
			Entries: entries,
		})
		s.emit(err, time.Now().Sub(start))
		if err == nil {
			log.Printf("Insert responded in %v", time.Now().Sub(start))
		} else {
			log.Printf("Insert errored with %v", err)
		}
	}
	s.print()
}

func cleanup(ctx context.Context, client pb.Service) {
	_, err := client.DeleteEvents(ctx, &pb.DeleteEventsRequest{
		Origin: benchmarkOrigin,
	})
	if err != nil {
		log.Fatalf("Failed to cleanup: %v", err)
	}
}

type summary struct {
	errors    int
	latencies []float64
}

func (s *summary) emit(err error, lat time.Duration) {
	if err != nil {
		s.errors++
		return
	}
	s.latencies = append(s.latencies, float64(lat)/(1000*1000)) // in ms
}

func (s *summary) print() {
	fmt.Println("")
	data := stats.Float64Data(s.latencies)

	for _, t := range []struct {
		name       string
		percentile float64
	}{
		{
			name:       "50th percentile",
			percentile: 50.0,
		},
		{
			name:       "90th percentile",
			percentile: 90.0,
		},
		{
			name:       "90th percentile",
			percentile: 99.0,
		},
	} {
		p, err := stats.Percentile(data, t.percentile)
		if err != nil {
			fmt.Printf("Failed to calculate %v: %v\n", t.name, err)
			continue
		}
		fmt.Printf("%v: %vms\n", t.name, p)
	}
	fmt.Printf("Errors: %v\n", s.errors)
}

func newSummary(n int) *summary {
	return &summary{latencies: make([]float64, 0, n)}
}
