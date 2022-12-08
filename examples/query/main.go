package main

import (
	"context"
	"log"
	"net/http"

	pb "github.com/mykodev/myko/proto"
)

func main() {
	ctx := context.Background()

	client := pb.NewServiceJSONClient("http://localhost:6959", &http.Client{})

	// List SQL query events collected from site navigation.
	resp, err := client.Query(ctx, &pb.QueryRequest{
		Origin: "site_nav",
		Event:  "sql_query",
	})
	if err != nil {
		log.Fatalf("Cannot query by origin and event: %v", err)
	}
	for _, ev := range resp.Events {
		log.Printf("%v: %v%v", ev.Name, ev.Value, ev.Unit)
	}

	const traceID = "0af7651916cd43dd8448eb211c80319c"
	resp, err = client.Query(ctx, &pb.QueryRequest{
		TraceId: traceID,
	})
	if err != nil {
		log.Fatalf("Cannot query by trace: %v", err)
	}
	log.Printf("Events collected for trace = %q", traceID)
	for _, ev := range resp.Events {
		log.Printf("%v: %v%v", ev.Name, ev.Value, ev.Unit)
	}
}
