package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"unsafe"
)

// cellSort externally sorts the cells in descending altitude order.
// The sort is stable to keep adjacent cells together (not required,
// but might help performance).
// Returns a channel producing the sorted data.
func cellSort(r <-chan []cell) <-chan []cell {
	// Make a temp directory for the external sort.
	dir, err := ioutil.TempDir("", "prominenceAltitudeSort")
	if err != nil {
		log.Fatal(err)
	}
	// Note: this defer will delete the directory before we're
	// done reading back the contents.  That's ok, we keep all
	// the files we need to read open.  Some OSes might choke
	// on this behavior.
	defer os.Remove(dir)

	// Keep track of write buffers, one for each altitude
	files := map[height]*os.File{}
	wbufs := map[height]*bufio.Writer{}

	// Read all the data, write it to temporary files, one for
	// each altitude.
	for cslice := range r {
		for _, c := range cslice {
			w, ok := wbufs[c.z]
			if !ok {
				// Haven't seen this altitude before.
				// Allocate file for this altitude.
				name := filepath.Join(dir, fmt.Sprintf("z%d", c.z))
				f, err := os.Create(name)
				if err != nil {
					log.Fatal(err)
				}
				// Immediately remove the file, so that the containing
				// directory will get deleted upon completion or error.
				os.Remove(name)

				// Allocate buffer for writing.
				w = bufio.NewWriterSize(f, 1024)

				// Save file and buffered writer.
				files[c.z] = f
				wbufs[c.z] = w
			}
			var b [unsafe.Sizeof(point{})]byte
			*(*point)(unsafe.Pointer(&b)) = c.p
			_, err = w.Write(b[:])
			if err != nil {
				log.Fatal(err)
			}
		}
		chunkPool.Put(cslice)
	}

	// Compute descending altitude order.
	alts := make([]int, 0, len(wbufs))
	for h := range wbufs {
		alts = append(alts, int(h))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(alts)))

	// Make a channel and shove the sorted data into it.
	c := make(chan []cell, 1)
	go func() {
		var chunker cellChunker
		chunker.c = c
		for _, a := range alts {
			h := height(a)

			// Finish writing the temporary file.
			w := wbufs[h]
			delete(wbufs, h)
			w.Flush()
			f := files[h]
			delete(files, h)

			// Read temporary file, send cells out.
			if _, err := f.Seek(0, 0); err != nil {
				log.Fatal(err)
			}
			buf := bufio.NewReader(f)
			for {
				var b [unsafe.Sizeof(point{})]byte
				_, err := buf.Read(b[:])
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatal(err)
				}
				chunker.send(cell{*(*point)(unsafe.Pointer(&b)), h})
			}
			f.Close()
		}
		chunker.close()
	}()
	return c
}
