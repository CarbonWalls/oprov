package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/oprov/oprov/internal/config"
	"github.com/oprov/oprov/internal/ui"
)

func main() {
	var path string
	flag.StringVar(&path, "path", "", "config file to manage (default: global opencode config)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "oprov — manage opencode custom providers and API keys\n\n")
		fmt.Fprintf(os.Stderr, "usage: oprov [flags]\n\nflags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nkeys:\n  a add · e edit · d delete · c set key · x clear key · t test · tab detail · ? help · q quit\n")
	}
	flag.Parse()

	if path == "" {
		p, _ := config.ResolveConfigPath()
		path = p
	}

	st, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oprov: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(st), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "oprov: %v\n", err)
		os.Exit(1)
	}
}
