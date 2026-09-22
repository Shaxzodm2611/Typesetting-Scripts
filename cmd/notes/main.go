package main

import (
	"fmt"
	"os"

	"github.com/Shaxzodm2611/Typesetting-Scripts/internal/cli"
	"golang.org/x/term"
)

var version = "dev"

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(cli.Run(os.Args[1:], cwd, cli.IO{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
		Interactive: term.IsTerminal(int(os.Stdin.Fd())),
	}, version))
}
