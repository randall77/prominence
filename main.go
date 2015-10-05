package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

var formatPtr = flag.String("format", "test", "format of input file (test, noaa1, noaa16)")
var minPtr = flag.Float64("min", 100, "minimum prominence to display (meters)")
var tmpDirPtr = flag.String("tmpdir", "", "temporary directory for external sort")
var P = flag.Int("P", runtime.NumCPU(), "width of parallel processing")

func main() {
	flag.Parse()

	proffile, err := os.Create("cpu.out")
	if err != nil {
		log.Fatal(err)
	}
	err = pprof.StartCPUProfile(proffile)
	if err != nil {
		log.Fatal(err)
	}

	var data dataSet
	switch *formatPtr {
	case "test":
		data = simpleDataSet([]cell{
			{point{0, 0}, 5},
			{point{0, 1}, 6},
			{point{0, 2}, 7},
			{point{0, 3}, 6},
			{point{1, 0}, 5},
			{point{1, 1}, 8},
			{point{1, 2}, 3},
			{point{1, 3}, 4},
		})
	case "noaa1":
		data = noaa1(flag.Arg(0))
	case "noaa16":
		data = noaa16(flag.Arg(0))
	case "srtm3":
		data = srtm3(flag.Arg(0))
	default:
		panic("unknown format " + *formatPtr)
	}

	// Get a reader for all the sample points.
	r := data.Reader()

	// Wrap reader in sniffer that generates a PNG for the data set.
	r2 := make(chan []cell, 1)
	minx, maxx, miny, maxy, minz, maxz := data.Bounds()
	const W = 2000
	const H = 1000
	m := &image.Gray{Pix: make([]uint8, W*H), Stride: W, Rect: image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{W, H}}}
	go func() {
		for cslice := range r {
			for _, c := range cslice {
				x := (c.p.x - minx) * W / (maxx - minx)
				y := (c.p.y - miny) * H / (maxy - miny)
				z := uint8(64 + (c.z-minz)*(256-64)/(maxz-minz))
				if m.Pix[x+y*W] < z {
					m.Pix[x+y*W] = z
				}
			}
			r2 <- cslice
		}
		close(r2)
		w, err := os.Create("globe.png")
		if err != nil {
			log.Fatal(err)
		}
		png.Encode(w, m)
		w.Close()
	}()

	computeProminence(r2, minx, maxx, func(peak, col, dom cell, island bool) {
		prom := peak.z - col.z
		_, _, meters := data.Pos(cell{point{minx, miny}, prom})
		if meters < *minPtr {
			return
		}

		if island {
			fmt.Printf("prominence of %s is %4.0fm (to sea level)\n",
				locString(data, peak),
				meters)
		} else {
			fmt.Printf("prominence of %s is %4.0fm (key col %s to %s)\n",
				locString(data, peak),
				meters,
				locString(data, col),
				locString(data, dom))
		}
	})

	pprof.StopCPUProfile()
}

// locString returns a human-readable location string for c, like:
//   12°03'55"N   3°23'52"W  678m
func locString(d dataSet, c cell) string {
	x, y, z := d.Pos(c)
	s := ""
	if y >= 0 {
		s += deg(y) + "N"
	} else {
		s += deg(-y) + "S"
	}
	s += " "
	if x >= 0 {
		s += deg(x) + "E"
	} else {
		s += deg(-x) + "W"
	}
	s += " "
	s += fmt.Sprintf("%4.0fm", z)
	return s
}

func deg(x float64) string {
	d := int(x)
	x -= float64(d)
	x *= 60
	m := int(x)
	x -= float64(m)
	x *= 60
	s := int(x + .5)
	return fmt.Sprintf("%3d°%02d'%02d\"", d, m, s)
}
