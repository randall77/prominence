package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Importer for SRTM3 Data
// http://dds.cr.usgs.gov/srtm/version2_1/SRTM3

type srtm3 string

func (file srtm3) Bounds() (minx, maxx coord, miny, maxy coord, minz, maxz height) {
	return 0, 432000, 0, 216000, -499, 8849
}

func (file srtm3) Pos(c cell) (lat, long, height float64) {
	return float64(c.p.x)/1200 - 180, 90 - float64(c.p.y)/1200, float64(c.z)
}

func (file srtm3) Reader() <-chan []cell {
	// Put files to be loaded into a channel
	work := make(chan string)
	go func() {
		dir := string(file)
		continents, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Fatal(err)
		}
		for _, continent := range continents {
			if continent.Name() == "index.html" {
				continue
			}
			subdir := filepath.Join(dir, continent.Name())
			files, err := ioutil.ReadDir(subdir)
			if err != nil {
				log.Fatal(err)
			}
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".hgt.zip") {
					work <- filepath.Join(subdir, f.Name())
				}
			}
		}
		close(work)
	}()

	// Return channel
	c := make(chan []cell, *P)

	// Use P workers to process all the work
	var wg sync.WaitGroup
	wg.Add(*P)
	for i := 0; i < *P; i++ {
		go func() {
			for name := range work {
				log.Print("reading " + name)

				// Parse tile name
				var ns, ew string
				var n, e int
				fmt.Sscanf(path.Base(name), "%1s%d%1s%d", &ns, &n, &ew, &e)
				if ns == "S" {
					n = -n
				}
				if ew == "W" {
					e = -e
				}

				// Extract tile data from zip file
				z, err := zip.OpenReader(name)
				if err != nil {
					panic("openreader")
					log.Fatal(err)
				}
				for _, sf := range z.File {
					if sf.Name[0] == '.' {
						continue // Junk in N21E034.hgt.zip
					}
					f, err := sf.Open() // always a zip of a single file
					if err != nil {
						panic("open")
						log.Fatal(err)
					}
					b, err := ioutil.ReadAll(f)
					if err != nil {
						panic("readall")
						log.Fatal(err)
					}

					// Figure out where we start
					x := 1200 * (180 + e)
					y := 1200 * (90 - n)

					// Note: tiles are named by their lower left corner.  But the data starts in
					// the upper left corner.  Plus tiles have one row overlap.
					// Adjust for all of that.
					y -= 1200

					var chunker cellChunker
					chunker.c = c
					for i := 0; i < 1200; i++ {
						for j := 0; j < 1200; j++ {
							z := height(int16(int(b[0])<<8 + int(b[1])))
							b = b[2:]
							if z == 0 {
								continue // ocean
							}
							if z == -32768 {
								continue // data voids - is this the right thing to do?
							}
							chunker.send(cell{point{coord(x + j), coord(y + i)}, z})
						}
						b = b[2:] // tiles have 1201 columns - the last one is the first column of the next tile
					}
					f.Close()
					chunker.flush()
				}
				z.Close()
				log.Print("done reading " + name)
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	return c
}
