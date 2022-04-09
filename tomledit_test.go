// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package tomledit_test

import (
	"bytes"
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
	doEmit = flag.Bool("emit", false, "Emit formatted output to stdout")
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
	tests := []struct {
		desc, input string
		want        string
		edit        func(*tomledit.Document)
	}{
		{
			desc:  "replace global",
			input: "key = {'x.y' = 0}",
			want:  "key = []",
			edit: func(doc *tomledit.Document) {
				doc.First("key").Value.X = parser.Array(nil)
			},
		},
		{
			desc:  "replace inline",
			input: "key={x=true}",
			want:  "key = {x = [1]}",
			edit: func(doc *tomledit.Document) {
				doc.First("key", "x").Value = parser.MustValue(`[1]`)
			},
		},
		{
			desc:  "remove global",
			input: "x=5\ny=10\n[z]\nok=true",
			want:  "[z]\nok = true",
			edit: func(doc *tomledit.Document) {
				doc.Global = nil
			},
		},
		{
			desc:  "remove inline",
			input: "[top]\nx={a=1,c=2}\n",
			want:  "[top]\nx = {a = 1}",
			edit: func(doc *tomledit.Document) {
				doc.First("top", "x", "c").Remove()
			},
		},
		{
			desc:  "remove section",
			input: "# A\n[a]\na=true\n\n# B\n[b]\nb=false\n[c]\nc=true",
			want:  "# A\n[a]\na = true\n\n[c]\nc = true",
			edit: func(doc *tomledit.Document) {
				doc.First("b").Remove()
			},
		},
		{
			desc:  "permute sections",
			input: "# A\n[a]\na=true\n\n# B\n[b]\nb=true\n",
			want:  "# B\n[b]\nb = true\n\n# A\n[a]\na = true",
			edit: func(doc *tomledit.Document) {
				doc.Sections = append(doc.Sections[1:], doc.Sections[0])
			},
		},
		{
			desc:  "insert global mapping",
			input: "x=0",
			want:  "x = 0\ny = 19  # OK",
			edit: func(doc *tomledit.Document) {
				doc.Global.Items = append(doc.Global.Items, &parser.KeyValue{
					Name:  parser.Key{"y"},
					Value: parser.MustValue(`19 # OK`),
				})
			},
		},
		{
			desc:  "insert table mapping",
			input: "[x]\ny=5",
			want:  "[x]\ny = 5\nz = [36, 24, 36]  # only if she's 5'3\"",
			edit: func(doc *tomledit.Document) {
				tab := doc.Sections[0]
				tab.Items = append(tab.Items, &parser.KeyValue{
					Name:  parser.Key{"z"},
					Value: parser.MustValue(`[36,24,36]# only if she's 5'3"`),
				})
			},
		},
		{
			desc:  "insert inline mapping",
			input: "x={a=0}",
			want:  "x = {a = 0, b = 'apples'}",
			edit: func(doc *tomledit.Document) {
				kv := doc.First("x").KeyValue
				tab := kv.Value.X.(parser.Inline)
				tab = append(tab, &parser.KeyValue{
					Name:  parser.Key{"b"},
					Value: parser.MustValue(`'apples'`),
				})
				kv.Value.X = tab
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			doc := mustParse(t, test.input)
			test.edit(doc)

			var buf bytes.Buffer
			if err := tomledit.Format(&buf, doc); err != nil {
				t.Fatalf("Format: %v", err)
			}
			got := strings.TrimSpace(buf.String())
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Wrong output: (-want, +got)\n%s", diff)
			}
		})
	}
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
