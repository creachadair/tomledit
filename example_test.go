// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package tomledit_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
)

func ExampleParse() {
	doc, err := tomledit.Parse(strings.NewReader(`# Example config

verbose=true

# A commented section
[commented]
  x = 3    # line comment

  # a commented mapping
  y = true
`))
	if err != nil {
		log.Fatalf("Parse: %v", err)
	}

	// Scan through the parsed document printing out all the keys defined in it,
	// in their order of occurrence.
	doc.Scan(func(key parser.Key, _ *tomledit.Entry) bool {
		fmt.Println(key)
		return true
	})
	// Output:
	//
	// verbose
	// commented
	// commented.x
	// commented.y
}
