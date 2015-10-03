package main

import "sync"

// A type for chunking up the sending of cells over a channel

// A cellChunker gathers batches of cells to send over a []cell channel.
type cellChunker struct {
	buf []cell
	c   chan<- []cell
}

// send will send c over the underlying channel, eventually.
func (cc *cellChunker) send(c cell) {
	if len(cc.buf) == cap(cc.buf) {
		if len(cc.buf) > 0 {
			cc.c <- cc.buf
		}
		i := chunkPool.Get()
		if i != nil {
			cc.buf = i.([]cell)[:0]
		} else {
			cc.buf = make([]cell, 0, 1024)
		}
	}
	cc.buf = append(cc.buf, c)
}

// close makes sure all cells are sent, then closes the underlying channel.
func (cc *cellChunker) close() {
	if len(cc.buf) > 0 {
		cc.c <- cc.buf
	}
	close(cc.c)
}

// A pool of buffers
var chunkPool sync.Pool
