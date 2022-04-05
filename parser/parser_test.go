// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package parser_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/creachadair/tomledit/parser"
	"github.com/google/go-cmp/cmp"
)

var (
	keyValueType = reflect.TypeOf((*parser.KeyValue)(nil))
	headingType  = reflect.TypeOf((*parser.Heading)(nil))
	commentsType = reflect.TypeOf(parser.Comments(nil))
)

func TestItems(t *testing.T) {
	type result struct {
		VType  reflect.Type
		Source string
	}
	tests := []struct {
		input string
		want  []result
	}{
		// Blanks.
		{"", nil},
		{"    ", nil},
		{"\n\n\t   \n", nil},

		// Comments.
		{"# c1\n# c2\n\n", []result{
			{commentsType, "# c1\n# c2"},
		}},

		// Key-value mappings.
		//
		// - Basic values.
		{`x=3`, []result{{keyValueType, `x = 3`}}},
		{`x=-3.2e+19`, []result{{keyValueType, `x = -3.2e+19`}}},
		{`x=-inf`, []result{{keyValueType, `x = -inf`}}},
		{`x=true`, []result{{keyValueType, `x = true`}}},
		{`x=false`, []result{{keyValueType, `x = false`}}},
		{`x="foo"`, []result{{keyValueType, `x = "foo"`}}},
		{`x='bar'`, []result{{keyValueType, `x = 'bar'`}}},

		// - Date and time stamps.
		{`x=2021-01-06`, []result{{keyValueType, `x = 2021-01-06`}}},
		{`x=15:00:23.22`, []result{{keyValueType, `x = 15:00:23.22`}}},
		{`x=2021-01-06T15:00:23`, []result{{keyValueType, `x = 2021-01-06T15:00:23`}}},
		{`x=2021-01-06T15:00:23Z`, []result{{keyValueType, `x = 2021-01-06T15:00:23Z`}}},
		{`x=2021-01-06T15:00:23+03:30`, []result{{keyValueType, `x = 2021-01-06T15:00:23+03:30`}}},

		// - Multi-line strings.
		{`x="""baz\nquux\n"""`, []result{{keyValueType, `x = """baz\nquux\n"""`}}},
		{"x='''baz\nquux\n'''", []result{{keyValueType, "x = '''baz\nquux\n'''"}}},

		// - Arrays.
		{"x=[]", []result{{keyValueType, `x = []`}}},
		{"x=[\n5,\n\"c\",\n]\n", []result{{keyValueType, `x = [5, "c"]`}}},
		{"x = [\n# ignored\n1,2,3\n# ignored\n, ]\n", []result{{keyValueType, `x = [1, 2, 3]`}}},

		// - Inline tables.
		{"x={} # whatever, bro\n", []result{{keyValueType, `x = {}`}}},
		{"x={a=2,b=\"four\"}\n", []result{{keyValueType, `x = {a = 2, b = "four"}`}}},

		// - Compound keys.
		{`x . y = true`, []result{{keyValueType, `x.y = true`}}},
		{`a . "b c" . d='qq'`, []result{{keyValueType, `a."b c".d = 'qq'`}}},
		{`"string thing"=["ding","ding",      ] # whoa`, []result{
			{keyValueType, `"string thing" = ["ding", "ding"]`},
		}},

		// Headings.
		{`[ a . b . c ]`, []result{{headingType, `[a.b.c]`}}},
		{`[ a . '' . c ]`, []result{{headingType, `[a."".c]`}}},
		{`[ a . "b.d" . c ]`, []result{{headingType, `[a."b.d".c]`}}},
		{`[[ a . b . c ]]`, []result{{headingType, `[[a.b.c]]`}}},
	}
	for _, test := range tests {
		p := parser.New(strings.NewReader(test.input))

		items, err := p.Items()
		if err != nil {
			t.Errorf("Items: unexpected error: %v", err)
			t.Logf("Input:\n%s", test.input)
			continue
		}

		var got []result
		for _, itm := range items {
			got = append(got, result{
				VType:  reflect.TypeOf(itm),
				Source: fmt.Sprint(itm),
			})
		}

		if diff := cmp.Diff(test.want, got, cmp.Comparer(func(t1, t2 reflect.Type) bool {
			return t1 == t2
		})); diff != "" {
			t.Errorf("Items: (-want, +got)\n%s", diff)
			t.Logf("Input:\n%s", test.input)
		}
	}
}

func TestParseKey(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		const input = ` a . "b.c d" . e`
		key, err := parser.ParseKey(input)
		if err != nil {
			t.Fatalf("ParseKey(%q): %v", input, err)
		}
		if diff := cmp.Diff(parser.Key{"a", "b.c d", "e"}, key); diff != "" {
			t.Errorf("Incorrect key result: (+want, -got)\n%s", diff)
		}
	})

	t.Run("Bad", func(t *testing.T) {
		for _, in := range []string{"", "  ", `#nope`, `.garbage`, `extra stuff`} {
			key, err := parser.ParseKey(in)
			if err == nil {
				t.Errorf("ParseKey(%q): got %v, wanted error", in, key)
			}
		}
	})
}
