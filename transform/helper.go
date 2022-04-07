// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package transform

import (
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
