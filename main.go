package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
)

var formatPtr = flag.String("format", "test", "format of input file (test, noaa1, noaa16)")
var minPtr = flag.Float64("min", 100, "minimum prominence to display (meters)")

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
	default:
		panic("unknown format " + *formatPtr)
	}

	//printASCII(data)
	minx, _, miny, _, _, _ := data.Bounds()

	computeProminence(data.Reader(), func(peak, col, dom cell, island bool) {
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

// printASCII prints a 60 wide 30 high map of the data.
func printASCII(d dataSet) {
	minx, maxx, miny, maxy, minz, maxz := d.Bounds()

	var m [30][60]byte
	for y := 0; y < 30; y++ {
		for x := 0; x < 60; x++ {
			m[y][x] = ' '
		}
	}
	r := d.Reader()
	for {
		c, ok := r()
		if !ok {
			break
		}
		if c.p.x < minx || c.p.x >= maxx || c.p.y < miny || c.p.y >= maxy {
			log.Fatalf("bad cell %s\n", c)
		}
		x := (c.p.x - minx) * 60 / (maxx - minx)
		y := (c.p.y - miny) * 30 / (maxy - miny)
		z := 'a' + byte((c.z-minz)*26/(maxz-minz))
		if m[y][x] < z {
			m[y][x] = z
		}
	}
	for y := 0; y < 30; y++ {
		fmt.Println(string(m[y][:]))
	}
}

// locString returns a human-readable location string for c, like:
//   12.0376°N   3.8752°W  678m
func locString(d dataSet, c cell) string {
	x, y, z := d.Pos(c)
	s := ""
	if y >= 0 {
		s += fmt.Sprintf("%8.4f°N", y)
	} else {
		s += fmt.Sprintf("%8.4f°S", -y)
	}
	s += " "
	if x >= 0 {
		s += fmt.Sprintf("%8.4f°E", x)
	} else {
		s += fmt.Sprintf("%8.4f°W", -x)
	}
	s += " "
	s += fmt.Sprintf("%4.0fm", z)
	return s
}
