package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sanjaynagpal/plover/pkg/persistence"
	"github.com/sanjaynagpal/plover/pkg/pipeline"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create pipeline with capacity 5 and Block overflow strategy
	p := pipeline.NewPipeline(5, pipeline.OverflowBlock)

	// Attach a simple logging stage that simulates work
	p.Use(func(ctx context.Context, item any, next pipeline.NextFunc) error {
		// simulate processing
		fmt.Printf("[stage] processing item: %v\n", item)
		time.Sleep(100 * time.Millisecond)
		return next(ctx, item)
	})

	// Final consumer stage
	p.Use(func(ctx context.Context, item any, next pipeline.NextFunc) error {
		fmt.Printf("[consumer] consumed: %v\n", item)
		return nil
	})

	// Optional: wire a simple file-backed queue for resilience
	q, err := persistence.NewFileQueue("plover_queue.log")
	if err != nil {
		log.Fatalf("failed to open queue: %v", err)
	}
	defer q.Close()

	// Start pipeline
	if err := p.Start(ctx); err != nil {
		log.Fatalf("failed to start pipeline: %v", err)
	}
	defer p.Stop()

	// Enqueue some items (persisted) and publish them into pipeline
	go func() {
		for i := 0; i < 20; i++ {
			item := fmt.Sprintf("msg-%02d", i)
			if err := q.Enqueue(item); err != nil {
				log.Printf("enqueue error: %v", err)
			}
			// publish from queue into pipeline
			// In real system: a separate recoverer job would replay persisted items
			time.Sleep(30 * time.Millisecond)
		}
	}()

	// Replayer: simplify - read from file queue and publish into pipeline
	go func() {
		for {
			val, err := q.Dequeue()
			if err != nil {
				// queue empty -> wait then retry
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if val == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !p.TryPublish(ctx, val) {
				// backpressure, try blocking publish to ensure persistence is meaningful
				if err := p.Publish(ctx, val); err != nil {
					log.Printf("publish failed: %v", err)
				}
			}
		}
	}()

	// Let demo run for a while
	time.Sleep(5 * time.Second)
	fmt.Println("demo finished")
}
