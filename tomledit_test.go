// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package tomledit_test

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/creachadair/tomledit/transform"
	"github.com/google/go-cmp/cmp"
)

var (
	testInput = flag.String("input", "", "Test input file")
	doEmit    = flag.Bool("emit", false, "Emit formatted output to stdout")
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

func mustFormat(t *testing.T, doc *tomledit.Document, more ...string) {
	t.Helper()

	if *doEmit {
		t.Logf("Writing formatted output for %s %s", t.Name(), strings.Join(more, " "))
		if err := tomledit.Format(os.Stdout, doc); err != nil {
			t.Fatalf("Format failed: %v", err)
		}
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
		var keys []string
		doc.Scan(func(key parser.Key, elt *tomledit.Entry) bool {
			keys = append(keys, key.String())
			return true
		})

		// All the keys defined in the test table, in definition order.  This
		// must be updated if the test input changes.
		want := []string{
			"p", "p.q", "p.r",
			"first.table", "first.table.a", "first.table.a.b", "first.table.a.c",
			"first.table.fuss.budget", "first.table.fuss.budget.x",
			"first.table.x", "first.table.y", "first.table.z", "first.table.list",
			"second-table", "second-table.foo",
			"p", "p.q", "p.r", "p.r.s.t", // first array element
			"p", "p.q", // second array element
		}
		if diff := cmp.Diff(want, keys); diff != "" {
			t.Errorf("Scan reported the wrong keys: (-want, +got)\n%s", diff)
		}
	})

	t.Run("Find", func(t *testing.T) {
		const wantMatches = 3

		found := doc.Find("p")
		if len(found) != wantMatches {
			t.Errorf("Find: got %d matches, want %d", len(found), wantMatches)
		}
		t.Logf("Matches: %v", found)
	})
}

func TestEdit(t *testing.T) {
	doc := mustParse(t, testDoc)
	mustFormat(t, doc, "(original document)")

	// Edit some values.
	doc.First("first", "table", "z").Value.X = parser.Array(nil)

	// Remove all the top-level keys.
	doc.Global = nil

	// Remove an inline table entry.
	doc.First("first", "table", "a", "c").Remove()

	mustFormat(t, doc, "(round 1)")

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

	mustFormat(t, doc, "(round 2)")
}

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
	doc := mustParse(t, `
# Welcome

[empty]

# Topic of much discussion.
[alpha_bravo]
charlie_delta = 'echo'
foxtrot = 0
whisky = { tango = false }

[[x]]
a = 1

[[x]]
a = 2

[stale]
great_balls_of = "fire"

[quite.late]
white.rabbit=true
`)
	mustFormat(t, doc, "(original document)")

	p := transform.Plan{
		{
			Desc: "Convert snake_case to kebab-case",
			T:    transform.SnakeToKebab(),
		},
		{
			Desc: "Ensure absent key is present",
			T: transform.EnsureKey(
				parser.Key{"alpha-bravo"},
				&parser.KeyValue{
					Name:  parser.Key{"new", "item"},
					Value: parser.MustValue("true").WithComment("A new value"),
				},
			),
		},
		{
			Desc: "Ensure present key is not replaced",
			T: transform.EnsureKey(
				parser.Key{"alpha-bravo"},
				&parser.KeyValue{
					Name:  parser.Key{"foxtrot"},
					Value: parser.MustValue(`"xyzzy"`),
				},
			),
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
		{
			Desc: "Sort sections by name",
			T: transform.Func(func(_ context.Context, doc *tomledit.Document) error {
				transform.SortSectionsByName(doc.Sections)
				return nil
			}),
		},
	}
	t.Log("Applying transformation plan...")
	ctx := transform.WithLogWriter(context.Background(), os.Stderr)
	if err := p.Apply(ctx, doc); err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	mustFormat(t, doc, "(after transformation)")
}
