package main

// A simpleDataSet is a dataSet specified by a slice of cells.
//  It has trivial mappings to real-world coordinates.
type simpleDataSet []cell

func (data simpleDataSet) Bounds() (minx, maxx coord, miny, maxy coord, minz, maxz height) {
	minx = data[0].p.x
	maxx = minx
	miny = data[0].p.y
	maxy = miny
	minz = data[0].z
	maxz = minz
	for _, c := range data[1:] {
		if c.p.x < minx {
			minx = c.p.x
		}
		if c.p.x > maxx {
			maxx = c.p.x
		}
		if c.p.y < miny {
			miny = c.p.y
		}
		if c.p.y > maxy {
			maxy = c.p.y
		}
		if c.z < minz {
			minz = c.z
		}
		if c.z > maxz {
			maxz = c.z
		}
	}
	maxx++
	maxy++
	maxz++
	return
}
func (data simpleDataSet) Pos(c cell) (lat, long, height float64) {
	return float64(c.p.x), float64(c.p.y), float64(c.z)
}
func (data simpleDataSet) Reader() reader {
	return simpleReader(data)
}

// simpleReader returns a reader which returns cells from data.
func simpleReader(data []cell) reader {
	i := 0
	return func() (cell, bool) {
		if i == len(data) {
			return cell{}, false
		}
		i++
		return data[i-1], true
	}
}
