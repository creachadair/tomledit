// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package transform

import (
	"context"
	"fmt"
	"strings"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
)

// SnakeToKebab transforms all the key names in doc from snake_case to
// kebab-case. This transformation cannot fail.
func SnakeToKebab() Transform {
	return func(_ context.Context, doc *tomledit.Document) error {
		doc.Scan(func(key parser.Key, e *tomledit.Entry) bool {
			if e.IsSection() && !e.IsGlobal() {
				e.Heading.Name = snakeToKebabKey(e.Heading.Name)
			}
			if e.KeyValue != nil {
				e.KeyValue.Name = snakeToKebabKey(e.KeyValue.Name)
			}
			return true
		})
		return nil
	}
}

func snakeToKebabKey(key parser.Key) parser.Key {
	out := make(parser.Key, len(key))
	for i, elt := range key {
		out[i] = strings.ReplaceAll(elt, "_", "-")
	}
	return out
}

// Rename renames the section or mapping at oldKey to newKey, and reports
// whether the rename was successful. The mapping is not moved within the
// document, only its label is changed.
func Rename(oldKey, newKey parser.Key) Transform {
	return func(_ context.Context, doc *tomledit.Document) error {
		found := doc.First(oldKey...)
		if found == nil {
			return fmt.Errorf("old key %q not found", oldKey)
		} else if found.IsSection() {
			found.Section.Heading.Name = newKey
		} else {
			found.KeyValue.Name = newKey
		}
		return nil
	}
}

// Remove removes the section or mapping at key, and reports whether the
// removal was successful.
func Remove(key parser.Key) Transform {
	return func(_ context.Context, doc *tomledit.Document) error {
		tgt := doc.First(key...)
		if tgt == nil {
			return fmt.Errorf("key %q not found", key)
		}
		tgt.Remove()
		return nil
	}
}

// MoveKey moves the mapping at oldKey from its current location to be a child
// of rootKey with the new name newKey. It reports whether the key was moved.
func MoveKey(oldKey, rootKey, newKey parser.Key) Transform {
	return func(_ context.Context, doc *tomledit.Document) error {
		src := doc.First(oldKey...)
		if src == nil || !src.IsMapping() {
			return fmt.Errorf("no mapping found for Key %q", oldKey)
		}
		dst := doc.First(rootKey...)
		if dst == nil {
			return fmt.Errorf("root key %q not found", rootKey)
		}

		src.Remove()
		src.Name = newKey
		if dst.IsSection() {
			dst.Items = append(dst.Items, src.KeyValue)
		} else if dst.IsInline() {
			v := dst.Value.X.(parser.Inline)
			dst.Value.X = append(v, src.KeyValue)
		} else {
			return fmt.Errorf("target %q is not a table", newKey)
		}
		return nil
	}
}
