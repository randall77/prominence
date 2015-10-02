package main

import (
	"math/rand"
	"sort"
	"testing"
)

func TestCellSort(t *testing.T) {
	cells := []cell{
		{point{0, 0}, 5},
		{point{1, 1}, 3},
		{point{2, 2}, 7},
		{point{3, 3}, 2},
		{point{4, 4}, 8},
		{point{5, 5}, 2},
		{point{6, 6}, 1},
		{point{7, 7}, 2},
		{point{8, 8}, 7},
		{point{9, 9}, 9},
	}

	// sort using cellSort
	d := simpleDataSet(cells)
	var cells2 []cell
	r := cellSort(d)
	for {
		c, ok := r()
		if !ok {
			break
		}
		cells2 = append(cells2, c)
	}

	// sort using byDescendingAltitude
	sort.Stable(byDescendingAltitude(cells))

	for i := range cells {
		if cells[i] != cells2[i] {
			t.Errorf("bad sort %v %v", cells, cells2)
			break
		}
	}
}

func TestCellSortBig(t *testing.T) {
	rnd := rand.New(rand.NewSource(127))

	var cells []cell
	for i := 0; i < 100000; i++ {
		x := coord(rnd.Intn(1000))
		y := coord(rnd.Intn(1000))
		z := height(rnd.Intn(100)) // NOTE: mac default is 256 open files, bastards!
		cells = append(cells, cell{point{x, y}, z})
	}

	// sort using cellSort
	d := simpleDataSet(cells)
	var cells2 []cell
	r := cellSort(d)
	for {
		c, ok := r()
		if !ok {
			break
		}
		cells2 = append(cells2, c)
	}

	// sort using byDescendingAltitude
	sort.Stable(byDescendingAltitude(cells))

	for i := range cells {
		if cells[i] != cells2[i] {
			t.Errorf("bad sort %v %v", cells, cells2)
			break
		}
	}
}

type byDescendingAltitude []cell

func (a byDescendingAltitude) Len() int           { return len(a) }
func (a byDescendingAltitude) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDescendingAltitude) Less(i, j int) bool { return a[i].z > a[j].z }
