package format

import (
	"errors"
	"strings"

	pb "github.com/mykodev/myko/proto"
)

func Verify(e *pb.Entry) error {
	if e.Origin == "" {
		return errors.New("entry doesn't contain an origin")
	}
	if strings.ContainsRune(e.Origin, ':') {
		return errors.New("origin contains illegal characters")
	}
	if strings.ContainsRune(e.Target, ':') {
		return errors.New("target contains illegal characters")
	}
	for _, ev := range e.Events {
		if strings.ContainsRune(ev.Name, ':') {
			return errors.New("event name contains illegal characters")
		}
	}
	return nil
}
