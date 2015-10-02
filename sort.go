package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"unsafe"
)

// cellSort externally sorts the cells in descending altitude order.
// The sort is stable to keep adjacent cells together (not required,
// but might help performance).
// Returns a reader for the sorted data.
func cellSort(data dataSet) reader {
	_, _, _, _, minz, maxz := data.Bounds()

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

	// Make some files, one for each altitude
	files := make([]*os.File, maxz-minz)
	for z := minz; z < maxz; z++ {
		name := filepath.Join(dir, fmt.Sprintf("z%d", z))
		files[z-minz], err = os.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		// Immediately remove the file, so that the containing
		// directory will get deleted upon completion or error.
		os.Remove(name)
	}

	// Set up some buffered writers to each file
	wbuf := make([]bufWriter, maxz-minz)
	for z := minz; z < maxz; z++ {
		wbuf[z-minz].w = files[z-minz]
	}

	// Read all the data, write it to the correct file
	r := data.Reader()
	for {
		c, ok := r()
		if !ok {
			break
		}
		wbuf[c.z-minz].write(c.p)
	}

	// Flush write buffers.
	for z := minz; z < maxz; z++ {
		wbuf[z-minz].flush()
	}

	// Now allocate some readers for the same files.
	rbuf := make([]bufReader, maxz-minz)
	for z := minz; z < maxz; z++ {
		files[z-minz].Seek(0, 0)
		rbuf[z-minz].r = files[z-minz]
	}

	// Return a reader that sucks data out of
	// the readers from highest to lowest.
	return func() (cell, bool) {
		for {
			if len(rbuf) == 0 {
				return cell{}, false
			}
			r := &rbuf[len(rbuf)-1]
			p, ok := r.read()
			if ok {
				return cell{p, minz + height(len(rbuf)-1)}, true
			}
			rbuf = rbuf[:len(rbuf)-1]
		}
	}
}

type bufWriter struct {
	// buf[:i] contains valid data.
	buf [100]point
	i   int
	w   io.Writer // underlying writer
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
	_, err := w.w.Write(s)
	if err != nil {
		log.Fatal(err)
	}
	w.i = 0
}

type bufReader struct {
	// buf[i:j] contains valid data
	buf  [100]point
	i, j int
	r    io.Reader // underlying reader
}

func (r *bufReader) read() (point, bool) {
	if r.i == r.j {
		// Get more data.
		p := unsafe.Pointer(&r.buf[0])
		n := int(unsafe.Sizeof(r.buf))
		s := *(*[]byte)(unsafe.Pointer(&slice{p, n, n}))
		b, err := r.r.Read(s)
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
