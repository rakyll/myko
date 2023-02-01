package aggregator

import (
	"strings"

	pb "github.com/mykodev/myko/proto"
)

type Summer struct {
	cap    int
	events map[string]*pb.Event
}

func NewSummer(cap int) *Summer {
	return &Summer{cap: cap, events: make(map[string]*pb.Event, cap)}
}

func (s *Summer) Size() int {
	return len(s.events)
}

func (s *Summer) Add(target, origin string, ev *pb.Event) {
	key := key(target, origin, ev.Name)
	v, ok := s.events[key]
	if !ok {
		s.events[key] = ev
	} else {
		v.Value += ev.Value
		s.events[key] = v
	}
}

func (s *Summer) ForEach(fn func(target, origin string, event *pb.Event)) {
	for k, ev := range s.events {
		target, origin, _ := parseKey(k)
		fn(target, origin, ev)
	}
}

func (s *Summer) Reset() {
	s.events = make(map[string]*pb.Event, s.cap)
}

func key(target, origin, name string) string {
	return target + ":" + origin + ":" + name
}

func parseKey(key string) (target, origin, event string) {
	v := strings.Split(key, ":")
	return v[0], v[1], v[2]
}
