// Program tomledit provides basic command-line support for reading and
// modifying TOML files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/creachadair/atomicfile"
	"github.com/creachadair/command"
	"github.com/creachadair/tomledit"
	"github.com/creachadair/tomledit/parser"
	"github.com/creachadair/tomledit/transform"
)

func main() {
	var cfg settings
	root := &command.C{
		Name: filepath.Base(os.Args[0]),
		Usage: `[options] command [args...]
help [command/topic]`,
		Help: `Read or modify the contents of a TOML file.`,

		SetFlags: func(_ *command.Env, fs *flag.FlagSet) {
			fs.StringVar(&cfg.Path, "path", "", "Path of TOML file to process")
		},

		Commands: []*command.C{
			{
				Name:  "list",
				Usage: "<key> ...",
				Help: `List the keys of key-value mappings.

With no keys, all the key-value mappings defined in the file are listed.
Otherwise, only those mappings having the given prefix are listed.`,

				Run: func(env *command.Env, args []string) error {
					doc, err := env.Config.(*settings).loadDocument()
					if err != nil {
						return err
					}
					keys, err := parseKeys(args)
					if err != nil {
						return err
					}
					doc.Scan(func(key parser.Key, _ *tomledit.Entry) bool {
						if hasPrefixIn(key, keys) {
							fmt.Println(key)
						}
						return true
					})
					return nil
				},
			},
			{
				Name:  "print",
				Usage: "<key>",
				Help:  "Print the value of the first definition of a key.",

				Run: func(env *command.Env, args []string) error {
					if len(args) == 0 {
						return env.Usagef("missing required key argument")
					}
					key, err := parser.ParseKey(args[0])
					if err != nil {
						return fmt.Errorf("parsing key: %w", err)
					}
					doc, err := env.Config.(*settings).loadDocument()
					if err != nil {
						return err
					}
					first := doc.First(key...)
					if first == nil {
						return fmt.Errorf("key %q not found", key)
					} else if first.IsSection() {
						fmt.Println(first.Section.Heading.String())
					} else {
						fmt.Println(first.KeyValue.Value.String())
					}
					return nil
				},
			},
			{
				Name:  "set",
				Usage: "<key> <value>",
				Help:  "Set the value of an existing key.",

				Run: func(env *command.Env, args []string) error {
					if len(args) != 2 {
						return env.Usagef("required arguments are <key> <value>")
					}
					key, err := parser.ParseKey(args[0])
					if err != nil {
						return fmt.Errorf("parsing key: %w", err)
					}
					val, err := parser.ParseValue(args[1])
					if err != nil {
						return fmt.Errorf("invalid TOML value: %w", err)
					}
					cfg := env.Config.(*settings)
					doc, err := cfg.loadDocument()
					if err != nil {
						return err
					}
					found := doc.Find(key...)
					if len(found) == 0 {
						return fmt.Errorf("key %q not found", key)
					} else if len(found) > 1 {
						return fmt.Errorf("found %d definitions of key %q", len(found), key)
					} else if !found[0].IsMapping() {
						return fmt.Errorf("%q is not a key-value mapping", key)
					}
					found[0].KeyValue.Value = val
					return cfg.saveDocument(doc)
				},
			},
			{
				Name: "add",
				Usage: `<table> <key> <value>
<global-key> <value>`,
				Help: `Add a key-value mapping to the specified section.

If no table name is specified, a mapping is added to the global table.
Otherwise, the mapping is added to the specified table (which must exist).
An error is reported if the key already exists, unless -replace is set.`,

				SetFlags: func(env *command.Env, fs *flag.FlagSet) {
					cfg := env.Config.(*settings)
					fs.BoolVar(&cfg.Replace, "replace", false, "Replace an existing mapping if present")
					fs.StringVar(&cfg.Text, "comment", "", "Comment text to add to the mapping")
				},

				Run: func(env *command.Env, args []string) error {
					if len(args) < 2 || len(args) > 3 {
						return env.Usagef("wrong number of arguments")
					}
					key, err := parser.ParseKey(args[0])
					if err != nil {
						return fmt.Errorf("parsing key %q: %w", args[0], err)
					}
					val, err := parser.ParseValue(args[len(args)-1])
					if err != nil {
						return fmt.Errorf("parsing value: %w", err)
					}
					var section parser.Key
					if len(args) == 3 {
						section = key
						key, err = parser.ParseKey(args[1])
						if err != nil {
							return fmt.Errorf("parsing key %q: %w", args[1], err)
						}
					}

					cfg := env.Config.(*settings)
					doc, err := cfg.loadDocument()
					if err != nil {
						return err
					}
					table := transform.FindTable(doc, section...)
					if table == nil {
						return fmt.Errorf("table %q not found", section)
					}
					var block parser.Comments
					if cfg.Text != "" {
						block = parser.Comments{cfg.Text}
					}
					if !transform.InsertMapping(table.Section, &parser.KeyValue{
						Block: block,
						Name:  key,
						Value: val,
					}, cfg.Replace) {
						return fmt.Errorf("key %q exists (use -replace to replace it)", key)
					}
					return cfg.saveDocument(doc)
				},
			},
			command.HelpCommand(nil),
		},
	}
	command.RunOrFail(root.NewEnv(&cfg), os.Args[1:])
}

type settings struct {
	Path    string
	Replace bool
	Text    string
}

func (s *settings) loadDocument() (*tomledit.Document, error) {
	if s.Path == "" {
		return nil, errors.New("no input -path is set")
	}
	f, err := os.Open(s.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return tomledit.Parse(f)
}

func (s *settings) saveDocument(doc *tomledit.Document) error {
	if s.Path == "" {
		return errors.New("no output -path is set")
	}
	f, err := atomicfile.New(s.Path, 0600)
	if err != nil {
		return err
	}
	defer f.Cancel()
	if err := tomledit.Format(f, doc); err != nil {
		return fmt.Errorf("formatting output: %w", err)
	}
	return f.Close()
}

func parseKeys(args []string) ([]parser.Key, error) {
	var keys []parser.Key
	for _, arg := range args {
		key, err := parser.ParseKey(arg)
		if err != nil {
			return nil, fmt.Errorf("parsing key %q: %w", arg, err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func hasPrefixIn(needle parser.Key, keys []parser.Key) bool {
	for _, key := range keys {
		if key.IsPrefixOf(needle) {
			return true
		}
	}
	return len(keys) == 0
}
