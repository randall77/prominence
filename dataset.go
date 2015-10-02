package main

// A dataSet provides an interface to topography data
// about the world.  A dataset is conceptually a 2d grid
// of altitude samples.
// Samples are identified by their (dense) grid coordinates.
// Samples are adjacent if they differ by exactly 1 in
// exactly one coordinate.
// Samples which are at sea level do not need to be
// considered part of the dataSet.
type dataSet interface {
	// Bounds returns bounds on the returned cells.
	// minx <= x < maxx
	// miny <= y < maxy
	// minz <= z < maxz
	Bounds() (minx, maxx, miny, maxy coord, minz, maxz height)

	// Returns a reader which will return all samples in the data set.
	// Multiple calls to Reader return independent readers.
	Reader() reader

	// Pos converts from the internal integral coordinate system
	// to standard coordinates.  lat and long are in degrees.
	// height is in meters above sea level.
	// lat is degrees north of the equator (-90 to 90).
	// long is degrees east from the prime meridian (-180 to 180).
	Pos(c cell) (lat, long, height float64)
}

// A reader is an iterator over the sample data.  When there is more
// data, the reader returns another cell of data and true.  When there
// is no more data, it returns the zero cell and false.
type reader func() (cell, bool)
