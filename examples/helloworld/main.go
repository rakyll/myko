package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	pb "github.com/mykodev/myko/proto"
)

func main() {

	ctx := context.Background()

	const traceID = "0af7651916cd43dd8448eb211c80319c"

	client := pb.NewServiceJSONClient("http://localhost:6959", &http.Client{})
	_, err := client.InsertEvents(ctx, &pb.InsertEventsRequest{
		Entries: []*pb.Entry{
			{
				TraceId: traceID,
				Origin:  "create_user",
				Events: []*pb.Event{
					{
						Name:  "render",
						Unit:  "ms",
						Value: 2.9,
					},
				},
			},
			{
				TraceId: traceID,
				Origin:  "create_user",
				Events: []*pb.Event{
					{
						Name:  "sql_count",
						Unit:  "",
						Value: 1,
					},
					{
						Name:  "sql_latency",
						Unit:  "ms",
						Value: 10.4,
					},
					{
						Name:  "sql_count",
						Unit:  "",
						Value: 1,
					},
					{
						Name:  "sql_latency",
						Unit:  "ms",
						Value: 3.21,
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to insert events: %v", err)
	}

	resp, err := client.ListEvents(ctx, &pb.ListEventsRequest{
		TraceId: traceID,
		Origin:  "create_user",
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}

	for _, item := range resp.Events {
		fmt.Printf("%v: %v%v\n", item.Name, item.Value, item.Unit)
	}
}
