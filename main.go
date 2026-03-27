package main

import (
	"fmt"
	"os"

	"github.com/kylesnowschwartz/the-themer/cmd"

	// Adapter packages register themselves via init().
	_ "github.com/kylesnowschwartz/the-themer/adapter/bat"
	_ "github.com/kylesnowschwartz/the-themer/adapter/delta"
	_ "github.com/kylesnowschwartz/the-themer/adapter/fzf"
	_ "github.com/kylesnowschwartz/the-themer/adapter/ghostty"
	_ "github.com/kylesnowschwartz/the-themer/adapter/vscode"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
