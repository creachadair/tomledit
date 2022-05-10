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
				Name: "list",
				Help: "List all the keys defined in the file.",

				Run: func(env *command.Env, args []string) error {
					if len(args) != 0 {
						return env.Usagef("extra arguments after command")
					}
					doc, err := env.Config.(*settings).loadDocument()
					if err != nil {
						return err
					}
					doc.Scan(func(key parser.Key, _ *tomledit.Entry) bool {
						fmt.Println(key)
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
			command.HelpCommand(nil),
		},
	}
	command.RunOrFail(root.NewEnv(&cfg), os.Args[1:])
}

type settings struct {
	Path string
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
