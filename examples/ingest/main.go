package main

import (
	"context"
	"log"
	"net/http"

	pb "github.com/mykodev/myko/proto"
)

func main() {
	ctx := context.Background()

	const target = "cluster-demo"

	client := pb.NewServiceJSONClient("http://localhost:6959", &http.Client{})
	_, err := client.InsertEvents(ctx, &pb.InsertEventsRequest{
		Entries: []*pb.Entry{
			{
				Target: target,
				Origin: "create_user",
				Events: []*pb.Event{
					{
						Name:  "render_ms",
						Value: 2.9,
					},
				},
			},
			{
				Target: target,
				Origin: "create_user",
				Events: []*pb.Event{
					{
						Name:  "sql_count",
						Value: 1,
					},
					{
						Name:  "sql_latency_ms",
						Value: 10.4,
					},
					{
						Name:  "sql_count",
						Value: 1,
					},
					{
						Name:  "sql_latency_ms",
						Value: 3.21,
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to insert events: %v", err)
	}
}
