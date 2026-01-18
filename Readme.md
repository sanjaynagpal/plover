# plover

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
