// Copyright (C) 2022 Michael J. Fromberger. All Rights Reserved.

package scanner_test

import (
	"io"
	"strings"
	"testing"

	"github.com/creachadair/tomledit/scanner"
	"github.com/google/go-cmp/cmp"
)

func TestScanner(t *testing.T) {
	type result struct {
		// N.B. Fields exported to simplify cmp usage.
		Tok  scanner.Token
		Text string
	}
	tests := []struct {
		input string
		want  []result
	}{
		{"", nil},
		{"   \t   ", nil},
		{"  \n  \n\t", []result{{scanner.Newline, ""}, {scanner.Newline, ""}}},

		{"# complete comment\n", []result{{scanner.Comment, "# complete comment"}}},
		{"# EOF comment", []result{{scanner.Comment, "# EOF comment"}}},

		{`0`, []result{{scanner.Integer, "0"}}},
		{`100`, []result{{scanner.Integer, "100"}}},
		{`-256_512`, []result{{scanner.Integer, "-256_512"}}},
		{`0x0`, []result{{scanner.Integer, "0x0"}}},
		{`0b0`, []result{{scanner.Integer, "0b0"}}},
		{`0o0`, []result{{scanner.Integer, "0o0"}}},

		{`-0.6e-15`, []result{{scanner.Float, "-0.6e-15"}}},
		{`inf +inf -inf`, []result{{scanner.Float, "inf"}, {scanner.Float, "+inf"}, {scanner.Float, "-inf"}}},
		{`nan`, []result{{scanner.Float, "nan"}}},

		{`"" ''`, []result{{scanner.String, `""`}, {scanner.LString, `''`}}},
		{`"\"\\\""`, []result{{scanner.String, `"\"\\\""`}}},
		{`"""foo\nbar"""`, []result{{scanner.MString, `"""foo\nbar"""`}}},
		{`'''foo'''`, []result{{scanner.MLString, `'''foo'''`}}},
		{`"""\
I am a man of \
constant sorrow.
"""`, []result{{scanner.MString, "\"\"\"\\\nI am a man of \\\nconstant sorrow.\n\"\"\""}}},
		{"'''\nI've seen trouble\nall my days.\n\n'''", []result{
			{scanner.MLString, "'''\nI've seen trouble\nall my days.\n\n'''"},
		}},

		{`[table] [[array]]`, []result{
			{scanner.LTable, "["}, {scanner.Word, "table"}, {scanner.RTable, "]"},
			{scanner.LArray, "[["}, {scanner.Word, "array"}, {scanner.RArray, "]]"},
		}},

		{`a."b c".d`, []result{
			{scanner.Word, "a"},
			{scanner.Dot, "."},
			{scanner.String, `"b c"`},
			{scanner.Dot, "."},
			{scanner.Word, "d"},
		}},

		{`1999-12-31T23:59:59.99999Z 2000-01-01T00:00:00+01:00`, []result{
			{scanner.DateTime, "1999-12-31T23:59:59.99999Z"},
			{scanner.DateTime, "2000-01-01T00:00:00+01:00"},
		}},
		{`2020-11-02T18:30:01.9001 2021-01-06T17:25:00`, []result{
			{scanner.LocalDateTime, "2020-11-02T18:30:01.9001"},
			{scanner.LocalDateTime, "2021-01-06T17:25:00"},
		}},
		{`1985-07-26 15:23:04.155 01:01:05`, []result{
			{scanner.LocalDate, "1985-07-26"},
			{scanner.LocalTime, "15:23:04.155"},
			{scanner.LocalTime, "01:01:05"},
		}},

		{`[ foo."bar" ]
baz = "quux"
frob = 2021-12-01
`, []result{
			{scanner.LTable, "["}, {scanner.Word, "foo"}, {scanner.Dot, "."},
			{scanner.String, `"bar"`}, {scanner.RTable, "]"}, {scanner.Newline, ""},
			{scanner.Word, "baz"}, {scanner.Equal, "="}, {scanner.String, `"quux"`}, {scanner.Newline, ""},
			{scanner.Word, "frob"}, {scanner.Equal, "="}, {scanner.LocalDate, "2021-12-01"}, {scanner.Newline, ""},
		}},

		{`1 +2 -3.2 +6e-9, {two=three}, "four"`, []result{
			{scanner.Integer, "1"}, {scanner.Integer, "+2"},
			{scanner.Float, "-3.2"}, {scanner.Float, "+6e-9"}, {scanner.Comma, ","},
			{scanner.LInline, "{"},
			{scanner.Word, "two"}, {scanner.Equal, "="}, {scanner.Word, "three"},
			{scanner.RInline, "}"}, {scanner.Comma, ","},
			{scanner.String, `"four"`},
		}},
	}

	for _, test := range tests {
		var got []result
		s := scanner.New(strings.NewReader(test.input))
		for s.Next() == nil {
			got = append(got, result{
				Tok:  s.Token(),
				Text: string(s.Text()),
			})
		}
		if s.Err() != io.EOF {
			t.Errorf("Next failed: %v", s.Err())
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Input: %#q\nTokens: (-want, +got)\n%s", test.input, diff)
		}
	}
}
