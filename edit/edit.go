// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package edit implements buffered position-based editing of byte slices.
package edit

import (
	"fmt"
	"sort"
)

// A Buffer is a queue of edits to apply to a given byte slice.
type Buffer struct {
	old []byte
	q   edits
}

// An edit records a single text modification: change the bytes in [start,end) to new.
type edit struct {
	start int
	end   int
	new   string
	force bool
}

// An edits is a list of edits that is sortable by start offset, breaking ties by end offset.
type edits []edit

func (x edits) Len() int      { return len(x) }
func (x edits) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x edits) Less(i, j int) bool {
	// Earlier edits first.
	if x[i].start != x[j].start {
		return x[i].start < x[j].start
	}
	// Insert before delete/replace.
	if (x[i].end == x[i].start) != (x[j].end == x[j].start) {
		return x[i].end == x[i].start
	}
	// Force delete before other delete/replace.
	if x[i].force != x[j].force {
		return x[i].force
	}
	return false
}

// NewBuffer returns a new buffer to accumulate changes to an initial data slice.
// The returned buffer maintains a reference to the data, so the caller must ensure
// the data is not modified until after the Buffer is done being used.
func NewBuffer(data []byte) *Buffer {
	return &Buffer{old: data}
}

func (b *Buffer) Insert(pos int, new string)         { b.edit(pos, pos, new, false) }
func (b *Buffer) Delete(start, end int)              { b.edit(start, end, "", false) }
func (b *Buffer) ForceDelete(start, end int)         { b.edit(start, end, "", true) }
func (b *Buffer) Replace(start, end int, new string) { b.edit(start, end, new, false) }

func (b *Buffer) edit(start, end int, new string, force bool) {
	if end < start || start < 0 || end > len(b.old) {
		panic("invalid edit position")
	}
	b.q = append(b.q, edit{start: start, end: end, new: new, force: force})
}

// Bytes returns a new byte slice containing the original data
// with the queued edits applied.
func (b *Buffer) Bytes() []byte {
	// Sort edits by starting position and then by ending position.
	// Breaking ties by ending position allows insertions at point x
	// to be applied before a replacement of the text at [x, y).
	// Given multiple inserts at the same position, keep in order.
	sort.Stable(b.q)

	var new []byte
	offset := 0
	for i := 0; i < len(b.q); i++ {
		e := b.q[i]
		if e.start < offset {
			e0 := b.q[i-1]
			panic(fmt.Sprintf("overlapping edits: [%d,%d)->%q, [%d,%d)->%q", e0.start, e0.end, e0.new, e.start, e.end, e.new))
		}
		new = append(new, b.old[offset:e.start]...)
		if e.force {
			for i+1 < len(b.q) && !b.q[i+1].force && b.q[i+1].start < e.end {
				i++
			}
		}
		offset = e.end
		new = append(new, e.new...)
	}
	new = append(new, b.old[offset:]...)
	return new
}

// String returns a string containing the original data
// with the queued edits applied.
func (b *Buffer) String() string {
	return string(b.Bytes())
}
