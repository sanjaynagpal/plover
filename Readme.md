# Plover 

A Ktor-inspired data pipeline implemented in Go.

Design principles:
- Asynchronous stages connected via channels (backpressure through bounded channels)
- Pluggable overflow strategies (Block, DropOldest, DropNewest)
- Optional resilient persistence using a simple file-backed append-only queue
- Clean start/stop semantics and context propagation

Quickstart:
1. Build: `go build ./cmd/plover`
2. Run: `./plover` (example producer + pipeline + consumer runs for a short demo)

This project is a minimal reference implementation. For production use, consider a production-grade persistent queue (BoltDB, BadgerDB, SQLite) and more robust error handling/resume semantics.

Files:
- pkg/pipeline: core pipeline and backpressure
- pkg/persistence: simple file-backed append-only queue (demo only)
- cmd/plover: example CLI that demonstrates usage

```prompt
Read the design document for Data Pipeline described in 
https://github.com/sanjaynagpal/go-learn/wiki/Design-Specification:-Resilient-Persistence-and-Backpressure-Management, 
https://github.com/sanjaynagpal/go-learn/wiki/Ktor%E2%80%90Inspired-Data-Pipeline:-An-Architectural-Overview, 
https://github.com/sanjaynagpal/go-learn/wiki/Ktor%E2%80%90Inspired-Data-Pipeline:-Architectural-Design-Specification, 
https://github.com/sanjaynagpal/go-learn/wiki/Understanding-the-Multi%E2%80%90Phase-Data-Pipeline:-A-Step%E2%80%90by%E2%80%90Step-Journey

Check @ktorio/ktor 

Write a implementation for Data Pipeline. Create a new project called plover in https://github.com/sanjaynagpal for this implementation.
```

## Design Notes

Design notes and trade-offs

The pipeline is intentionally simple and single-worker. You can extend it with worker pools and multiple parallel executors. Ktor uses coroutine-based pipeline contexts and writer loops; here we model similar separation with channels and stage chaining.
Backpressure is modeled with a bounded channel and three strategies:
Block: block until space (safe but may apply backpressure to producers)
DropOldest: evict oldest items to keep throughput
DropNewest: drop incoming item when full
Persistence is a minimal append-only file queue. It demonstrates replay semantics; a production system should use an established embedded store (Bolt/Badger/SQLite) with offset/truncation management and concurrency-safe reads/writes.
The pipeline's Stage signature mirrors Ktor's interceptors: a function receives a subject and a "next" continuation to call to proceed.