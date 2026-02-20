// This program generates env config scripts from config.Config struct tags for
// unix and windows platforms.
// Usage: go run ./cmd/config_generator [platform] [filename] [value_tag]
// Example: go run ./cmd/config_generator unix settings.basic.env basic
package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/pchchv/go-icq/config"
)

var platformKeywords = map[string]struct {
	comment    string
	assignment string
}{
	"windows": {
		comment:    "rem ",
		assignment: "set ",
	},
	"unix": {
		comment:    "# ",
		assignment: "export ",
	},
}

func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: go run cmd/config_generator [platform] [filename] [value_tag]")
		os.Exit(1)
	}

	platform := args[0]
	keywords, ok := platformKeywords[platform]
	if !ok {
		fmt.Fprintf(os.Stderr, "unable to find platform `%s`\n", platform)
		os.Exit(1)
	}

	filename := args[1]
	fmt.Println("writing to", filename)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating file: %s\n", err.Error())
		os.Exit(1)
	}

	defer f.Close()

	valueTag := args[2]
	configType := reflect.TypeOf(config.Config{})
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		// check if field is optional and has empty value
		// use the specified value tag
		val := field.Tag.Get(valueTag)
		// skip optional fields with empty values
		if field.Tag.Get("required") == "false" && val == "" {
			continue
		}

		comment := field.Tag.Get("description")
		if err := writeComment(f, comment, 80, keywords.comment); err != nil {
			fmt.Fprintf(os.Stderr, "error writing to file: %s\n", err.Error())
			os.Exit(1)
		}

		varName := field.Tag.Get("envconfig")
		if err := writeAssignment(f, keywords.assignment, varName, val); err != nil {
			fmt.Fprintf(os.Stderr, "error writing to file: %s\n", err.Error())
			os.Exit(1)
		}
	}
}

func writeAssignment(w io.Writer, keyword string, varName string, val string) (err error) {
	_, err = fmt.Fprintf(w, "%s%s=%s\n\n", keyword, varName, val)
	return
}

func writeComment(w io.Writer, comment string, width uint, keyword string) error {
	// adjust wrapping threshold to accommodate comment keyword length
	width = width - uint(len(keyword))
	comment = wordwrap.WrapString(comment, width)
	// prepend lines with comment keyword
	comment = strings.ReplaceAll(comment, "\n", fmt.Sprintf("\n%s", keyword))
	_, err := fmt.Fprintf(w, "%s%s\n", keyword, comment)
	return err
}
