package format

import (
	"strings"

	pb "github.com/mykodev/myko/proto"
)

func escapeString(v string) string {
	return strings.ReplaceAll(v, ":", "_")
}

func Espace(e *pb.Entry) *pb.Entry {
	e.Origin = escapeString(e.Origin)
	e.TraceId = escapeString(e.TraceId)
	for _, ev := range e.Events {
		ev.Name = escapeString(ev.Name)
		ev.Unit = escapeString(ev.Unit)
	}
	return e
}
