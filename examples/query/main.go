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
		log.Printf("%v: %v", ev.Name, ev.Value)
	}

	const target = "cluster-demo"
	resp, err = client.Query(ctx, &pb.QueryRequest{
		Target: target,
	})
	if err != nil {
		log.Fatalf("Cannot query by target: %v", err)
	}
	log.Printf("Events collected for target = %q", target)
	for _, ev := range resp.Events {
		log.Printf("%v: %v", ev.Name, ev.Value)
	}
}
