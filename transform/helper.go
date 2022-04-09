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
		return keyLess(nameOf(ss[i]), nameOf(ss[j]))
	})
}

func nameOf(s *tomledit.Section) parser.Key {
	if s.Heading == nil {
		return nil
	}
	return s.Heading.Name
}

func keyLess(k1, k2 parser.Key) bool {
	i, j := 0, 0
	for i < len(k1) && j < len(k2) {
		if k1[i] < k2[j] {
			return true
		}
		i++
		j++
	}
	return i == len(k1) && j < len(k2)
}
