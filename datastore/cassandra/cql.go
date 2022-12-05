package cassandra

import (
	"errors"
	"fmt"
	"strings"
)

type Filter struct {
	TraceID string
	Origin  string
	Event   string
}

func (f Filter) CQL() (string, error) {
	if f.TraceID == "" && f.Origin == "" && f.Event == "" {
		return "", errors.New("no trace_id, origin or event")
	}

	// TODO: Escape the input.
	var filters []string
	if f.TraceID != "" {
		filters = append(filters, fmt.Sprintf("trace_id = '%v'", f.TraceID))
	}
	if f.Origin != "" {
		filters = append(filters, fmt.Sprintf("origin = '%v'", f.Origin))
	}
	if f.Event != "" {
		filters = append(filters, fmt.Sprintf("event = '%v'", f.Event))
	}

	return "WHERE " + strings.Join(filters, " AND "), nil
}
