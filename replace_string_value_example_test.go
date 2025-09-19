// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package tomledit_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/transform"
)

func Example() {
	data := []byte(`
	A = "Va"

	[ B ]
	C = "Vc"

	[ D.E ]
	F = "Vf"
	`)
	replaceList := []struct {
		path  []string
		value string
	}{
		{[]string{"A"}, "Va2"},
		{[]string{"B", "C"}, "Vc2"},
		{[]string{"D", "E", "F"}, "Vf2"},
	}
	doc, err := tomledit.Parse(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("Parse: %v", err)
	}
	for _, r := range replaceList {
		entry := doc.First(r.path...)
		// entry != nil should be checked here
		// entry.KeyValue != nil should be checked here

		// key = last element in path
		key := r.path[len(r.path)-1]

		// Create new document to replace fragment in original one
		snippet := fmt.Sprintf("%s = %q\n", key, r.value)
		newDoc, err := tomledit.Parse(strings.NewReader(snippet))
		if err != nil {
			log.Fatalf("Parse: %v", err)
		}
		newItem := newDoc.First(key)

		transform.InsertMapping(entry.Section, newItem.KeyValue, true)
	}

	var buf bytes.Buffer
	if err := tomledit.Format(&buf, doc); err != nil {
		log.Fatalf("Format: %v", err)
	}
	fmt.Println(buf.String())
	// Output:
	// A = "Va2"
	//
	// [B]
	// C = "Vc2"
	//
	// [D.E]
	// F = "Vf2"
}
