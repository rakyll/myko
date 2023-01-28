package kusto

import (
	"context"
	"encoding/json"
	"io"
	"log"

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

type KustoEntry struct {
	TraceID string  `json:"trace_id,omitempty"`
	Origin  string  `json:"origin,omitempty"`
	Event   string  `json:"event,omitempty"`
	Value   float64 `json:"value,omitempty"`
}

func (s *Session) IngestAll(ctx context.Context, entries []*KustoEntry) error {
	r, w := io.Pipe()
	go func() {
		defer w.Close()

		encoder := json.NewEncoder(w)
		for _, e := range entries {
			if err := encoder.Encode(e); err != nil {
				log.Printf("Failed to encode %v: %v", e, err)
			}
		}
	}()
	result, err := s.client.FromReader(ctx, r, ingest.FileFormat(ingest.MultiJSON))
	if err == nil {
		err = <-result.Wait(ctx) // TODO: Block when closing and handle retryable errors.
	}
	return err
}

func (s *Session) Close() error {
	return s.client.Close()
}
