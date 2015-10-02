package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"strings"
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

func (file srtm3) Reader() reader {
	dir := string(file)
	continents, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var tiles []string
	for _, c := range continents {
		subdir := filepath.Join(dir, c.Name())
		files, err := ioutil.ReadDir(subdir)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".hgt.zip") {
				tiles = append(tiles, filepath.Join(subdir, f.Name()))
			}
		}
	}

	buf := make([]cell, 0, 1200*1200)
	return func() (cell, bool) {
		for {
			if len(buf) > 0 {
				c := buf[0]
				buf = buf[1:]
				return c, true
			}
			if len(tiles) == 0 {
				return cell{}, false // no more tiles, all done
			}
			t := tiles[0]
			tiles = tiles[1:]
			log.Print("reading " + t)

			// Parse tile name
			var ns, ew string
			var n, e int
			fmt.Sscanf(path.Base(t), "%1s%d%1s%d", &ns, &n, &ew, &e)
			if ns == "S" {
				n = -n
			}
			if ew == "W" {
				e = -e
			}

			// Extract tile data from zip file
			z, err := zip.OpenReader(t)
			if err != nil {
				log.Fatal(err)
			}
			defer z.Close()
			f, err := z.File[0].Open() // always a zip of a single file
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			b, err := ioutil.ReadAll(f)
			if err != nil {
				log.Fatal(err)
			}

			// Figure out where we start
			x := 1200 * (180 + e)
			y := 1200 * (90 - n)

			// Note: tiles are named by their lower left corner.  But the data starts in
			// the upper left corner.  Plus tiles have one row overlap.
			// Adjust for all of that.
			y -= 1200

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
					buf = append(buf, cell{point{coord(x + j), coord(y + i)}, z})
				}
				b = b[2:] // tiles have 1201 columns - the last one is the first column of the next tile
			}
		}
	}
}
