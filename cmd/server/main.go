package main

import (
	"flag"
	"fmt"
	"os"
)

// default build fields populated by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	showHelp := flag.Bool("help", false, "Display help")
	showVersion := flag.Bool("version", false, "Display build information")

	flag.Parse()

	switch {
	case *showVersion:
		fmt.Printf("%-10s %s\n", "version:", version)
		fmt.Printf("%-10s %s\n", "commit:", commit)
		fmt.Printf("%-10s %s\n", "date:", date)
		os.Exit(0)
	case *showHelp:
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func main() {}
