package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/cli"
)

func main() {
	if err := cli.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
