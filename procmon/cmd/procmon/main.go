package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func usage() {
	fmt.Fprintf(os.Stderr, `procmon %s — per-process resource collector

Usage:
  procmon collect [--data-dir DIR] [--docker-data-root PATH]
                                     Take one sample of all processes
                                     (docker-data-root auto-detected from
                                     /etc/docker/daemon.json by default)
  procmon prune   [--data-dir DIR] [--keep-days N]
                                     Delete data files older than N days (default 7)
  procmon version                    Print version
  procmon help                       Print this help
`, version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "collect":
		os.Exit(runCollect(os.Args[2:]))
	case "prune":
		os.Exit(runPrune(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
