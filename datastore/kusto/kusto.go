package kusto

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/ingest"
	"github.com/mykodev/myko/config"
)

type Session struct {
	client    *ingest.Streaming
	tableName string
}

func NewSession(dataConfig config.DataConfig) (*Session, error) {
	kConfig := dataConfig.KustoConfig

	csb := kusto.NewConnectionStringBuilder(kConfig.Endpoint).WithAzCli()
	kustoClient, err := kusto.New(csb)
	if err != nil {
		return nil, err
	}

	client, err := ingest.NewStreaming(kustoClient, kConfig.Database, kConfig.Table)
	if err != nil {
		return nil, err
	}
	return &Session{
		client:    client,
		tableName: kConfig.Table,
	}, nil
}

type Entry struct {
	Target string  `json:"target,omitempty"`
	Origin string  `json:"origin,omitempty"`
	Event  string  `json:"event,omitempty"`
	Value  float64 `json:"value,omitempty"`
}

func (s *Session) IngestAll(ctx context.Context, entries []*Entry) error {
	r, w := io.Pipe()

	errCh := make(chan error)
	go func() {
		defer w.Close()

		encoder := json.NewEncoder(w)
		for _, e := range entries {
			if err := encoder.Encode(e); err != nil {
				errCh <- err
			}
		}
		close(errCh)
	}()
	result, err := s.client.FromReader(ctx, r, ingest.FileFormat(ingest.MultiJSON))
	if err != nil {
		return err
	}

	select {
	case err = <-result.Wait(ctx):
		return err
	case err = <-errCh:
		return err
	}
}

func (s *Session) Close() error {
	return s.client.Close()
}
