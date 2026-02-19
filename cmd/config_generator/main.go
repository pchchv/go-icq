package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: go run cmd/config_generator [platform] [filename] [value_tag]")
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
}

func writeAssignment(w io.Writer, keyword string, varName string, val string) (err error) {
	_, err = fmt.Fprintf(w, "%s%s=%s\n\n", keyword, varName, val)
	return
}
