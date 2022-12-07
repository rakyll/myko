package aggregator

import (
	"strings"

	pb "github.com/mykodev/myko/proto"
)

type Summer struct {
	cap    int
	events map[string]*pb.Event
}

func NewSum(cap int) *Summer {
	return &Summer{cap: cap, events: make(map[string]*pb.Event, cap)}
}

func (s *Summer) Size() int {
	return len(s.events)
}

func (s *Summer) Add(traceID, origin string, ev *pb.Event) {
	key := key(traceID, origin, ev.Name, ev.Unit)
	v, ok := s.events[key]
	if !ok {
		s.events[key] = ev
	} else {
		v.Value += ev.Value
		s.events[key] = v
	}
}

func (s *Summer) ForEach(fn func(traceID, origin string, event *pb.Event) error) error {
	for k, ev := range s.events {
		traceID, origin, _, _ := parseKey(k)
		return fn(traceID, origin, ev)
	}
	return nil
}

func (s *Summer) Reset() {
	s.events = make(map[string]*pb.Event, s.cap)
}

func key(traceID, origin, name, unit string) string {
	return traceID + ":" + origin + ":" + name + ":" + unit
}

func parseKey(key string) (traceID, origin, name, unit string) {
	v := strings.Split(key, ":")
	return v[0], v[1], v[2], v[3]
}
