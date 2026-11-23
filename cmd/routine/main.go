// Command routine is a terminal app for a dynamic routine: a priority queue of
// activities plus a daily work quota tracked in slices, with an encrypted state
// you can keep in a synced folder. Run "routine help".
package main

import (
	"fmt"
	"os"

	"github.com/DiegoAndradeD/dynamic-routine/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
