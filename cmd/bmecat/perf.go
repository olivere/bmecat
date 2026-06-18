package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/olivere/bmecat"
)

// perfCommand roughly evaluates the Read performance of the bmecat package.
// It reads both BMEcat 1.2 and 2005 files via the version-neutral facade.
type perfCommand struct {
	header           *bmecat.Header
	progress         bool
	numProducts      uint32
	numCatalogGroups uint32
	numClassifGroups uint32
	completed        atomic.Uint32
}

func init() {
	RegisterCommand("perf", func(flags *flag.FlagSet) Command {
		cmd := new(perfCommand)
		flags.BoolVar(&cmd.progress, "P", false, "Print progress")
		return cmd
	})
}

func (cmd *perfCommand) Describe() string {
	return "Performance tester"
}

func (cmd *perfCommand) Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s perf [-P] <file>\n", os.Args[0])
}

func (cmd *perfCommand) Run(args []string) error {
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
	start := time.Now()
	err = bmecat.NewReader(f, o...).Do(ctx, cmd)
	if err != nil {
		return err
	}
	took := time.Since(start)
	if cmd.progress {
		fmt.Println()
	}
	if cmd.header == nil {
		return errors.New("did not receive HEADER")
	}

	fmt.Printf("%-24s: %7s\n", "Version", cmd.header.Version)
	fmt.Printf("%-24s: %7d / %7d\n", "Products", cmd.header.NumberOfProducts, cmd.numProducts)
	fmt.Printf("%-24s: %7d / %7d\n", "Catalog Groups", cmd.header.NumberOfCatalogGroups, cmd.numCatalogGroups)
	fmt.Printf("%-24s: %7d / %7d\n", "Classification Groups", cmd.header.NumberOfClassificationGroups, cmd.numClassifGroups)
	fmt.Printf("%-24s: %v\n", "Took", took.String())
	fmt.Printf("%-24s: %7.2f\n", "Products/sec", float64(cmd.numProducts)/took.Seconds())

	return nil
}

func (cmd *perfCommand) HandleHeader(header *bmecat.Header) error {
	cmd.header = header
	return nil
}

func (cmd *perfCommand) HandleCatalogGroup(c *bmecat.CatalogGroup) error {
	atomic.AddUint32(&cmd.numCatalogGroups, 1)
	return nil
}

func (cmd *perfCommand) HandleClassificationGroup(c *bmecat.ClassificationGroup) error {
	atomic.AddUint32(&cmd.numClassifGroups, 1)
	return nil
}

func (cmd *perfCommand) HandleProduct(p *bmecat.Product) error {
	atomic.AddUint32(&cmd.numProducts, 1)
	return nil
}

func (cmd *perfCommand) HandleComplete() {
	cmd.completed.Add(1)
}
