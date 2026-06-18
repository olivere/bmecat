package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/olivere/bmecat"
)

// infoCommand parses the BMEcat header and prints the information found there.
// It reads both BMEcat 1.2 and 2005 files via the version-neutral facade.
type infoCommand struct {
	header   *bmecat.Header
	progress bool
}

func init() {
	RegisterCommand("info", func(flags *flag.FlagSet) Command {
		cmd := new(infoCommand)
		flags.BoolVar(&cmd.progress, "P", false, "Print progress")
		return cmd
	})
}

func (cmd *infoCommand) Describe() string {
	return "Print BMEcat information"
}

func (cmd *infoCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s info [-P] <file>\n", os.Args[0])
}

func (cmd *infoCommand) Run(args []string) error {
	ctx := context.Background()

	if len(args) == 0 {
		return errors.New("missing file name")
	}

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	var o []bmecat.ReaderOption
	if cmd.progress {
		f := func(pass int, offset int64) {
			fmt.Printf("Pass %d, Offset %6d kB\r", pass, offset/1024)
		}
		o = append(o, bmecat.WithReaderProgress(f))
	}
	err = bmecat.NewReader(f, o...).Do(ctx, cmd)
	if err != nil {
		return err
	}
	if cmd.progress {
		fmt.Println()
	}

	if cmd.header == nil {
		return errors.New("did not receive HEADER")
	}

	fmt.Printf("%-24s: %7s\n", "Version", cmd.header.Version)
	fmt.Printf("%-24s: %7d\n", "Products", cmd.header.NumberOfProducts)
	fmt.Printf("%-24s: %7d\n", "Catalog Groups", cmd.header.NumberOfCatalogGroups)
	fmt.Printf("%-24s: %7d\n", "Classification Groups", cmd.header.NumberOfClassificationGroups)

	return nil
}

func (cmd *infoCommand) HandleHeader(header *bmecat.Header) error {
	cmd.header = header
	return io.EOF
}
