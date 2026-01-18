package persistence

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// FileQueue is a simple append-only file-backed queue.
// This is a minimal demo persistence for resilience. For production use a robust DB.
type FileQueue struct {
	fpath  string
	f      *os.File
	mu     sync.Mutex
	readR  *bufio.Reader
	writeW *bufio.Writer
}

// NewFileQueue opens/creates an append-only queue file.
func NewFileQueue(path string) (*FileQueue, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	// move write pointer to end
	if _, err := f.Seek(0, 2); err != nil {
		f.Close()
		return nil, err
	}
	return &FileQueue{
		fpath:  path,
		f:      f,
		readR:  bufio.NewReader(f),
		writeW: bufio.NewWriter(f),
	}, nil
}

// Enqueue appends an item as JSON line.
func (q *FileQueue) Enqueue(item any) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if _, err := q.f.Write(append(b, '\n')); err != nil {
		return err
	}
	return q.f.Sync()
}

// Dequeue reads the next item from the file. This simple implementation
// reads sequentially from the beginning; it is primarily for demo replays.
// For real workloads you would maintain offsets and truncation or use a DB.
func (q *FileQueue) Dequeue() (any, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	// try read a line
	line, err := q.readR.ReadBytes('\n')
	if err != nil {
		// EOF or other -> no data
		return nil, errors.New("empty")
	}
	var v any
	if err := json.Unmarshal(line, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// Close flushes and closes file handles.
func (q *FileQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.writeW != nil {
		_ = q.writeW.Flush()
	}
	return q.f.Close()
}
