package main

import "sync"

// A cellChunker gathers batches of cells to send over a []cell channel.
type cellChunker struct {
	buf []cell
	c   chan<- []cell
}

// send will send c over the underlying channel, eventually.
func (cc *cellChunker) send(c cell) {
	buf := cc.buf
	if len(buf) == cap(buf) {
		if len(buf) > 0 {
			cc.c <- buf
		}
		i := chunkPool.Get()
		if i != nil {
			buf = i.([]cell)[:0]
		} else {
			buf = make([]cell, 0, 1024)
		}
	}
	cc.buf = append(buf, c)
}

// close makes sure all cells are sent, then closes the underlying channel.
func (cc *cellChunker) close() {
	if len(cc.buf) > 0 {
		cc.c <- cc.buf
	}
	close(cc.c)
}

// A pool of unused buffers
var chunkPool sync.Pool
