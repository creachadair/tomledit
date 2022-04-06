// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package tomledit_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/creachadair/tomledit/transform"
)

func mustParse(t *testing.T, s string) *tomledit.Document {
	t.Helper()
	doc, err := tomledit.Parse(strings.NewReader(s))
	if err != nil {
		t.Logf("Input:\n%s", s)
		t.Fatalf("Parse failed: %v", err)
	}
	return doc
}

func mustFormat(t *testing.T, doc *tomledit.Document) {
	t.Helper()
	fmt.Println("--- formatted output for", t.Name())
	var out tomledit.Formatter
	if err := out.Format(os.Stdout, doc); err != nil {
		t.Fatalf("Format failed: %v", err)
	}
}

const testDoc = `
# free

# top-level mapping
p = { q = [], r = {}}

# bound
[ first . table ]
  # Various spacing shenanigans.
  a = { b = 3, c = [4,5, # ppp
        6,
  ] } # qqq

  # Compound keys and values.
  fuss . budget = {x = true} # barbaric yawp

  x = 14  # hey what's up
  y = 'three'
# A complex value.
z = [4, 5, # whatever
    ['a', 'b', # hark
          'c' # hey 
        , 'd'], # foob
      6, 7] #hate

list = [
10, 20, 30, 40,
]

[second-table]
foo = 'bar'

# A repeated table array.
[[p]]
q = 1
r = { s.t = 'u'} # v

[[p]]
q = 2

# free comment again
`

func TestFormat(t *testing.T) {
	mustFormat(t, mustParse(t, testDoc))
}

func TestScan(t *testing.T) {
	doc := mustParse(t, testDoc)
	t.Run("All", func(t *testing.T) {
		doc.Scan(func(key parser.Key, elt *tomledit.Entry) bool {
			t.Logf("Key %q: %v", key, elt)
			return true
		})
	})

	t.Run("Filtered", func(t *testing.T) {
		want := parser.Key{"p"}
		var found []*tomledit.Entry
		doc.Scan(func(key parser.Key, elt *tomledit.Entry) bool {
			if elt.IsSection() && key.Equals(want) {
				found = append(found, elt)
			}
			return true
		})

		t.Logf("Found %d matches for key %q", len(found), want)
		for i, elt := range found {
			t.Logf("[%d]: %v", i+1, elt)
		}
	})
}

func TestEdit(t *testing.T) {
	doc := mustParse(t, testDoc)

	// Edit some values.
	doc.First("first", "table", "z").Value.X = parser.Array(nil)

	// Remove all the top-level keys.
	doc.Global = nil

	// Remove an inline table entry.
	doc.First("first", "table", "a", "c").Remove()

	mustFormat(t, doc)

	// Move the first section to the end.
	doc.Sections = append(doc.Sections[1:], doc.Sections[0])

	// Remove a section by name.
	doc.First("first", "table").Remove()

	// Delete some matching keys.
	found := doc.Find("p")
	if len(found) == 0 {
		t.Fatal("No entries matching p")
	}
	t.Logf("Deleting %d entries matching p", len(found))
	for _, elt := range found {
		if !elt.Remove() {
			t.Errorf("Remove %q failed", elt)
		}
		// Verify that deleting a second time does not succeed
		if elt.Remove() {
			t.Errorf("Remove %q succeeded twice", elt)
		}
	}

	// Add a mapping to the last section.
	v := parser.MustValue(`"fenchurch street station"`)
	s := doc.Sections[len(doc.Sections)-1]
	s.Items = append(s.Items, &parser.KeyValue{
		Block: parser.Comments{
			"An additional item added programmatically.",
			"Don't let it go to your head",
		},
		Name:  parser.Key{"left", "luggage", "office"},
		Value: v,
	}, parser.Comments{"# A final wave goodbye"})

	mustFormat(t, doc)
}

var testInput = flag.String("input", "", "Test input file")

func TestData(t *testing.T) {
	if *testInput == "" {
		t.Skip("Skipping because -input file is not set")
	}

	f, err := os.Open(*testInput)
	if err != nil {
		t.Fatalf("Opening test input: %v", err)
	}
	defer f.Close()

	doc, err := tomledit.Parse(f)
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	mustFormat(t, doc)
}

func TestTransform(t *testing.T) {
	doc := mustParse(t, `[alpha_bravo]
charlie_delta = 'echo'
foxtrot = 0
whisky = { tango = false }

[empty]

[stale]
great_balls_of = "fire"
`)
	p := transform.Plan{
		{
			Desc: "Convert snake_case to kebab-case",
			T:    transform.SnakeToKebab(),
		},
		{
			Desc: "Rename section",
			T: transform.Rename(
				parser.Key{"alpha-bravo"},
				parser.Key{"charlie", "fox trot"},
			),
		},
		{
			Desc: "Rename inline key",
			T: transform.Rename(
				parser.Key{"charlie", "fox trot", "whisky", "tango"},
				parser.Key{"epsilon"},
			),
		},
		{
			Desc: "Move item to a new location",
			T: transform.MoveKey(
				parser.Key{"stale", "great-balls-of"},
				parser.Key{"empty"},
				parser.Key{"horking-great-balls-of"},
			),
		},
		{
			Desc: "Rename now-non-empty section",
			T: transform.Rename(
				parser.Key{"empty"},
				parser.Key{"non-empty"},
			),
		},
		{
			Desc: "Remove stale section",
			T:    transform.Remove(parser.Key{"stale"}),
		},
	}
	ctx := transform.WithLogWriter(context.Background(), os.Stderr)
	if err := p.Apply(ctx, doc); err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	mustFormat(t, doc)
}
