package main

import (
	"context"
	"log"

	"github.com/janmarkuslanger/godata/etl"
)

func main() {
	ctx := context.Background()

	pipeline := etl.New("basic").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{"value": 1},
				{"value": 2},
			}, nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			record["value"] = record["value"].(int) + 10
			return record, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			log.Printf("records: %v", records)
			return nil
		})

	if err := pipeline.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
