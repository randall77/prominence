package main

import (
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
// Returns a reader for the sorted data.
func cellSort(r reader) reader {
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
	wbufs := map[height]*bufWriter{}

	// Read all the data, write it to temporary files, one for
	// each altitude.
	for {
		c, ok := r()
		if !ok {
			break
		}
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
			w = &bufWriter{f: f}
			wbufs[c.z] = w
		}
		w.write(c.p)
	}

	// Compute descending altitude order.
	alts := make([]int, 0, len(wbufs))
	for h := range wbufs {
		alts = append(alts, int(h))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(alts)))

	// Return a reader that sucks data out of
	// the files from highest to lowest altitude.
	var r2 bufReader
	var h height
	return func() (cell, bool) {
		for {
			p, ok := r2.read()
			if ok {
				return cell{p, h}, true
			}

			// Close temp file, we're done with it.
			if r2.f != nil {
				r2.f.Close()
			}

			// Pick next altitude.
			if len(alts) == 0 {
				return cell{}, false
			}
			h = height(alts[0])
			alts = alts[1:]

			// Finish writing temporary file.
			w := wbufs[h]
			delete(wbufs, h)
			w.flush()
			f := w.f

			// Start reading temporary file.
			if _, err := f.Seek(0, 0); err != nil {
				log.Fatal(err)
			}
			r2.f = f
		}
	}
}

type bufWriter struct {
	// buf[:i] contains valid data.
	buf [100]point
	i   int
	f   *os.File // underlying file
}

func (w *bufWriter) write(p point) {
	if w.i == len(w.buf) {
		w.flush()
	}
	w.buf[w.i] = p
	w.i++
}

type slice struct {
	ptr unsafe.Pointer
	len int
	cap int
}

func (w *bufWriter) flush() {
	if w.i == 0 {
		return
	}
	p := unsafe.Pointer(&w.buf[0])
	n := w.i * int(unsafe.Sizeof(point{}))
	s := *(*[]byte)(unsafe.Pointer(&slice{p, n, n}))
	_, err := w.f.Write(s)
	if err != nil {
		log.Fatal(err)
	}
	w.i = 0
}

type bufReader struct {
	// buf[i:j] contains valid data
	buf  [1000]point
	i, j int
	f    *os.File // underlying file
}

func (r *bufReader) read() (point, bool) {
	if r.i == r.j {
		// Get more data.
		if r.f == nil {
			return point{}, false
		}
		p := unsafe.Pointer(&r.buf[0])
		n := int(unsafe.Sizeof(r.buf))
		s := *(*[]byte)(unsafe.Pointer(&slice{p, n, n}))
		b, err := r.f.Read(s)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if b == 0 {
			return point{}, false
		}
		if b%int(unsafe.Sizeof(point{})) != 0 {
			log.Fatalf("partial point read")
		}
		r.i = 0
		r.j = b / int(unsafe.Sizeof(point{}))
	}
	p := r.buf[r.i]
	r.i++
	return p, true
}
