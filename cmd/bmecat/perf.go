package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/olivere/bmecat/v12"
)

// perfCommand roughly evaluates the Read performance of the bmecat package.
type perfCommand struct {
	header           *v12.Header
	progress         bool
	numArticles      uint32
	numCatalogGroups uint32
	numClassifGroups uint32
	completed        uint32
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

	var o []v12.ReaderOption
	if cmd.progress {
		f := func(pass int, offset int64) {
			fmt.Printf("Pass %d, Offset %6d kB\r", pass, offset/1024)
		}
		o = append(o, v12.WithReaderProgress(f))
	}
	start := time.Now()
	err = v12.NewReader(f, o...).Do(ctx, cmd)
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

	fmt.Printf("%-24s: %7d / %7d\n", "Products", cmd.header.NumberOfArticles, cmd.numArticles)
	fmt.Printf("%-24s: %7d / %7d\n", "Catalog Groups", cmd.header.NumberOfCatalogGroups, cmd.numCatalogGroups)
	fmt.Printf("%-24s: %7d / %7d\n", "Classification Groups", cmd.header.NumberOfClassificationGroups, cmd.numClassifGroups)
	fmt.Printf("%-24s: %v\n", "Took", took.String())
	fmt.Printf("%-24s: %7.2f\n", "Products/sec", float64(cmd.numArticles)/took.Seconds())

	return nil
}

func (cmd *perfCommand) HandleHeader(header *v12.Header) error {
	cmd.header = header
	return nil
}

func (cmd *perfCommand) HandleCatalogGroup(c *v12.CatalogGroup) error {
	atomic.AddUint32(&cmd.numCatalogGroups, 1)
	return nil
}

func (cmd *perfCommand) HandleClassificationGroup(c *v12.ClassificationGroup) error {
	atomic.AddUint32(&cmd.numClassifGroups, 1)
	return nil
}

func (cmd *perfCommand) HandleArticle(article *v12.Article) error {
	atomic.AddUint32(&cmd.numArticles, 1)
	return nil
}

func (cmd *perfCommand) HandleComplete() {
	atomic.AddUint32(&cmd.completed, 1)
}
