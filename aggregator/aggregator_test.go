package aggregator

import (
	"fmt"
	"testing"

	pb "github.com/mykodev/myko/proto"
)

func TestSummer(t *testing.T) {
	tests := []struct {
		name        string
		entries     []*pb.Entry
		wantEntries []*pb.Entry
		wantSize    int
	}{
		{
			name:        "empty",
			entries:     []*pb.Entry{},
			wantEntries: []*pb.Entry{},
			wantSize:    0,
		},
		// TODO: Add a case with no trace IDs.
		{
			name: "no trace",
			entries: []*pb.Entry{
				{
					Origin: "origin_1",
					Events: []*pb.Event{
						{Name: "name_1", Value: 10},
						{Name: "name_1", Value: 10},
						{Name: "name_1", Value: 10},
					},
				},
			},
			wantEntries: []*pb.Entry{
				{
					Origin: "origin_1",
					Events: []*pb.Event{
						{Name: "name_1", Value: 30},
					},
				},
			},
			wantSize: 1,
		},
		{
			name: "basic",
			entries: []*pb.Entry{
				{
					TraceId: "trace_1",
					Origin:  "origin_1",
					Events: []*pb.Event{
						{Name: "name_1", Value: 10},
						{Name: "name_1", Value: 50},
						{Name: "name_1", Value: 100},
					},
				},
				{
					TraceId: "trace_1",
					Origin:  "origin_2",
					Events: []*pb.Event{
						{Name: "name_2", Value: 200},
					},
				},
				{
					TraceId: "trace_1",
					Origin:  "origin_2",
					Events: []*pb.Event{
						{Name: "name_2", Value: 200},
						{Name: "name_1", Value: 500},
					},
				},
			},
			wantEntries: []*pb.Entry{
				{
					TraceId: "trace_1",
					Origin:  "origin_1",
					Events: []*pb.Event{
						{Name: "name_1", Value: 160},
					},
				},
				{
					TraceId: "trace_1",
					Origin:  "origin_2",
					Events: []*pb.Event{
						{Name: "name_1", Value: 500},
					},
				},
				{
					TraceId: "trace_1",
					Origin:  "origin_2",
					Events: []*pb.Event{
						{Name: "name_2", Value: 400},
					},
				},
			},
			wantSize: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSummer(256)
			for _, e := range tt.entries {
				for _, ev := range e.Events {
					s.Add(e.TraceId, e.Origin, ev)
				}
			}
			if size := s.Size(); size != tt.wantSize {
				t.Errorf("Size  = %v, wantSize %v", size, tt.wantSize)
			}
			for _, wantEntry := range tt.wantEntries {
				for _, wantEvent := range wantEntry.Events {
					if !s.exists(wantEntry.TraceId, wantEntry.Origin, wantEvent) {
						t.Errorf("Can't find the event: %v", wantEvent)
					}
				}
			}
		})
	}
}

func BenchmarkSummer(b *testing.B) {
	const (
		originCardinality = 10
		eventCardinality  = 20
	)

	entries := make([]*pb.Entry, 0, originCardinality)
	for i := 0; i < originCardinality; i++ {
		ev := &pb.Entry{
			TraceId: "xxx",
			Origin:  fmt.Sprintf("origin_%d", i),
		}
		ev.Events = make([]*pb.Event, 0, eventCardinality)
		for j := 0; j < eventCardinality; j++ {
			ev.Events = append(ev.Events, &pb.Event{
				Name:  fmt.Sprintf("event_%d", j),
				Value: 1.0,
			})
		}
		entries = append(entries, ev)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summer := NewSummer(1024)
		for _, entry := range entries {
			for _, ev := range entry.Events {
				summer.Add(entry.TraceId, entry.Origin, ev)
			}
		}
	}
}

func (s *Summer) exists(traceID, origin string, ev *pb.Event) bool {
	key := key(traceID, origin, ev.Name)
	v, ok := s.events[key]
	if !ok {
		return false
	}
	if v.Name != ev.Name {
		return false
	}
	if v.Value != ev.Value {
		return false
	}
	return true
}
