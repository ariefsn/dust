package main

import (
	"fmt"
	"os"

	"github.com/ariefsn/dust/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
