package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	wtdetach "github.com/Konboi/git-wt-detach"
)

var version = "0.1.3"

func main() {
	cli := wtdetach.CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("git-wt-detach"),
		kong.Description("Temporarily detach a branch checked out in another worktree."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
	)

	if err := cli.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "✖ %s\n", err)
		ctx.Exit(1)
	}
}
