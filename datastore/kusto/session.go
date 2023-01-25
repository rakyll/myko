package kusto

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/ingest"
	"github.com/mykodev/myko/config"
	pb "github.com/mykodev/myko/proto"
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

type kustoEntry struct {
	TraceID   string    `json:"trace_id,omitempty"`
	Origin    string    `json:"origin,omitempty"`
	Event     string    `json:"event,omitempty"`
	Value     float64   `json:"value,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func (s *Session) Ingest(ctx context.Context, traceID, origin string, ev *pb.Event) error {
	body, err := json.Marshal(kustoEntry{
		TraceID:   traceID,
		Origin:    origin,
		Event:     ev.Name,
		Value:     ev.Value,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	result, err := s.client.FromReader(ctx, bytes.NewBuffer(body),
		ingest.Table(s.tableName),
		ingest.FileFormat(ingest.JSON),
	)
	if err == nil {
		err = <-result.Wait(ctx) // TODO: Block when closing and handle retryable errors.
	}
	return err
}

func (s *Session) Close() error {
	return s.client.Close()
}
