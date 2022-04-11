// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package transform

import (
	"sort"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
)

// FindTable returns the entry the first table with the given name in doc, or
// nil if no such table exists. An empty name denotes the global table.
func FindTable(doc *tomledit.Document, name ...string) *tomledit.Entry {
	if len(name) == 0 {
		if doc.Global == nil {
			return nil
		}
		return &tomledit.Entry{Section: doc.Global}
	}
	key := parser.Key(name)
	for _, s := range doc.Sections {
		if s.Name.Equals(key) {
			return &tomledit.Entry{Section: s}
		}
	}
	return nil
}

// SortSectionsByName performs a stable in-place sort of the given slice of
// sections by their name.
func SortSectionsByName(ss []*tomledit.Section) {
	sort.SliceStable(ss, func(i, j int) bool {
		return ss[i].TableName().Before(ss[j].TableName())
	})
}

// SortKeyValuesByName performs a stable in-place sort of items, so that any
// key-value entries are ordered by their names, but other items such as
// comments are left in their original positions.
func SortKeyValuesByName(items []parser.Item) {
	s := subseq{orig: items}
	for i, item := range items {
		kv, ok := item.(*parser.KeyValue)
		if ok {
			s.pos = append(s.pos, i)
			s.name = append(s.name, kv.Name)
		}
	}

	sort.Stable(s)
}

// subseq implements sort.Interface to sort a subsequence of the elements of
// the original slice.
//
// To do this, it maintains a hash table of the offsets in the original slice
// where the elements to be sorted are stored, then "sorts" the indices of the
// hash table with comparison and swap functions that indirect through to the
// underlying values.
//
// For efficiency, it also caches a positionally-mapped slice of the keys to be
// sorted, to avoid the overhead of repeatedly loading and type-asserting the
// original values out of their interface wrappers.
type subseq struct {
	orig []parser.Item // the original input slice
	pos  []int         // pos[i] is the offset in orig of the ith subsequence item
	name []parser.Key  // the key of the current ith subsequence item
}

func (s subseq) Len() int           { return len(s.pos) }
func (s subseq) Less(i, j int) bool { return s.name[i].Before(s.name[j]) }

func (s subseq) Swap(i, j int) {
	// N.B. we do not permute s.pos, because the offsets in the original
	// sequence where the values are stored do not change, only the contents at
	// those offsets.
	oi, oj := s.pos[i], s.pos[j]
	s.orig[oi], s.orig[oj] = s.orig[oj], s.orig[oi]
	s.name[i], s.name[j] = s.name[j], s.name[i]
}
