// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package transform

import (
	"sort"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
)

// FindTable returns the entry the first table with the given name in doc, or
// nil if no such table exists.
func FindTable(doc *tomledit.Document, name ...string) *tomledit.Entry {
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
		return nameOf(ss[i]).Before(nameOf(ss[j]))
	})
}

func nameOf(s *tomledit.Section) parser.Key {
	if s.Heading == nil {
		return nil
	}
	return s.Heading.Name
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
// the original slice. It builds mirror slices of the names and indices of the
// items to be sorted in the original slice, then sorts these. When swapping
// elements of the mirror slices, the corresponding elements of the original
// are also swapped.
type subseq struct {
	orig []parser.Item // the original input slice
	pos  []int         // offset in orig of the ith subsequence item
	name []parser.Key  // the key of the current ith subsequence item
}

func (s subseq) Len() int           { return len(s.pos) }
func (s subseq) Less(i, j int) bool { return s.name[i].Before(s.name[j]) }

func (s subseq) Swap(i, j int) {
	oi, oj := s.pos[i], s.pos[j]
	s.orig[oi], s.orig[oj] = s.orig[oj], s.orig[oi]
	s.name[i], s.name[j] = s.name[j], s.name[i]
}
