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
	for _, event := range e.Events {
		event.Name = escapeString(event.Name)
		event.Unit = escapeString(event.Unit)
	}
	return e
}
