package pipeline

import (
	"context"
	"errors"
	"sync"
	"time"
)

// OverflowStrategy defines how the pipeline handles over-capacity publishes.
type OverflowStrategy int

const (
	OverflowBlock OverflowStrategy = iota
	OverflowDropOldest
	OverflowDropNewest
)

// NextFunc is called to continue pipeline execution.
type NextFunc func(ctx context.Context, item any) error

// Stage is an interceptor/worker in the pipeline.
type Stage func(ctx context.Context, item any, next NextFunc) error

// Pipeline is the central struct.
type Pipeline struct {
	capacity int
	strat    OverflowStrategy

	stages []Stage

	ch    chan any
	wg    sync.WaitGroup
	start sync.Once
	stop  sync.Once

	mu      sync.RWMutex
	closing bool
}

// NewPipeline creates a pipeline with given capacity and overflow strategy.
func NewPipeline(capacity int, strat OverflowStrategy) *Pipeline {
	if capacity <= 0 {
		capacity = 1
	}
	return &Pipeline{
		capacity: capacity,
		strat:    strat,
		stages:   make([]Stage, 0),
		ch:       make(chan any, capacity),
	}
}

// Use adds a stage to the pipeline. Stages run in the order they are added.
func (p *Pipeline) Use(s Stage) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, s)
}

// Start launches the pipeline workers.
func (p *Pipeline) Start(ctx context.Context) error {
	var err error
	p.start.Do(func() {
		// single worker that executes pipeline; could be extended to multiple
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case it, ok := <-p.ch:
					if !ok {
						return
					}
					// Execute pipeline chain
					p.execute(ctx, it)
				}
			}
		}()
	})
	return err
}

// Stop stops accepting new items and waits for processing to finish.
func (p *Pipeline) Stop() {
	p.mu.Lock()
	if p.closing {
		p.mu.Unlock()
		return
	}
	p.closing = true
	close(p.ch)
	p.mu.Unlock()

	p.wg.Wait()
}

// Publish blocks or returns error based on overflow strategy.
func (p *Pipeline) Publish(ctx context.Context, item any) error {
	p.mu.RLock()
	if p.closing {
		p.mu.RUnlock()
		return errors.New("pipeline is closed")
	}
	p.mu.RUnlock()

	switch p.strat {
	case OverflowBlock:
		select {
		case p.ch <- item:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case OverflowDropOldest:
		for {
			select {
			case p.ch <- item:
				return nil
			default:
				// drop oldest to make space
				select {
				case <-p.ch:
				default:
				}
			}
		}
	case OverflowDropNewest:
		select {
		case p.ch <- item:
			return nil
		default:
			// drop the new item
			return errors.New("dropped newest item due to overflow")
		}
	default:
		return errors.New("unknown overflow strategy")
	}
}

// TryPublish attempts a non-blocking publish; returns true if accepted.
func (p *Pipeline) TryPublish(ctx context.Context, item any) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closing {
		return false
	}
	select {
	case p.ch <- item:
		return true
	default:
		return false
	}
}

// execute runs the chain of stages for a single item.
func (p *Pipeline) execute(ctx context.Context, item any) {
	index := 0
	var next NextFunc
	next = func(ctx context.Context, it any) error {
		if index >= len(p.stages) {
			return nil
		}
		// capture current stage and increment index for next call
		s := p.stages[index]
		index++
		return s(ctx, it, next)
	}
	// run the pipeline with a short bounded context to ensure progress
	// (could be configured)
	_ = next(ctx, item)
}

// Convenience: Publish with default background context and blocking semantics.
func (p *Pipeline) PublishBG(item any) error {
	return p.Publish(context.Background(), item)
}

// Wait utility: wait with timeout for pipeline to drain (simple helper).
func (p *Pipeline) Wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
