package main

import (
	"io/ioutil"
	"log"
	"os"
	"sort"
	"sync"
	"unsafe"
)

const bufSize = 1024

type fileRange struct {
	off int64
	len int
}

// cellSort externally sorts the cells in descending altitude order.
// Returns a channel producing the sorted data.
func cellSort(r <-chan []cell) <-chan []cell {
	// Make a temp file for the external sort.
	f, err := ioutil.TempFile(*tmpDirPtr, "prominenceAltitudeSort")
	if err != nil {
		log.Fatal(err)
	}
	// Remove the file to keep the filesystem clean.
	// Note: this call deletes the file before we've even used it.
	// That's ok, we keep the file open.  At least most OSes do
	// the right thing here.
	os.Remove(f.Name())

	// Keep track of which file ranges hold data from each altitude.
	var lock sync.Mutex
	ranges := map[height][]fileRange{}

	// Read all the cells, write them to the temporary file in groups
	// of identical altitude points.
	var wg sync.WaitGroup
	wg.Add(*P)
	for i := 0; i < *P; i++ {
		go func() {
			// Keep a write buffer for each altitude.
			wbufs := map[height]*wbuf{}
			for cslice := range r {
				for _, c := range cslice {
					w := wbufs[c.z]
					if w == nil {
						w = &wbuf{}
						wbufs[c.z] = w
					}
					if w.n == len(w.buf) {
						// Write full buffer to the temp file.
						b := int(unsafe.Sizeof(w.buf))
						s := *(*[]byte)(unsafe.Pointer(&slice{unsafe.Pointer(&w.buf), b, b}))
						lock.Lock()
						off, err := f.Seek(0, 1)
						if err != nil {
							log.Fatal(err)
						}
						_, err = f.Write(s)
						if err != nil {
							log.Fatal(err)
						}
						ranges[c.z] = append(ranges[c.z], fileRange{off, b})
						lock.Unlock()
						w.n = 0
					}
					w.buf[w.n] = c.p
					w.n++
				}
				chunkPool.Put(cslice)
			}
			// Write any remaining parital buffers to the temp file.
			for h, w := range wbufs {
				b := w.n * int(unsafe.Sizeof(point{}))
				s := *(*[]byte)(unsafe.Pointer(&slice{unsafe.Pointer(&w.buf), b, b}))
				lock.Lock()
				off, err := f.Seek(0, 1)
				if err != nil {
					log.Fatal(err)
				}
				_, err = f.Write(s)
				if err != nil {
					log.Fatal(err)
				}
				ranges[h] = append(ranges[h], fileRange{off, b})
				lock.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// Compute descending altitude order.
	alts := make([]int, 0, len(ranges))
	for h := range ranges {
		alts = append(alts, int(h))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(alts)))

	// Make a channel and shove the sorted data into it.
	c := make(chan []cell, 1)
	go func() {
		var points [bufSize]point
		var chunker cellChunker
		chunker.c = c
		for _, a := range alts {
			h := height(a)

			// Read chunks from temporary file.
			for _, rng := range ranges[h] {
				b := rng.len
				if b > int(unsafe.Sizeof(points)) {
					log.Fatal("block too big")
				}
				s := *(*[]byte)(unsafe.Pointer(&slice{unsafe.Pointer(&points), b, b}))
				_, err := f.ReadAt(s, rng.off)
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < b/int(unsafe.Sizeof(point{})); i++ {
					chunker.send(cell{points[i], h})
				}
			}
		}
		chunker.flush()
		close(c)
	}()
	return c
}

type wbuf struct {
	buf [bufSize]point
	n   int
}

type slice struct {
	p unsafe.Pointer
	l int
	c int
}
