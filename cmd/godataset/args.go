package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type arguments struct {
	outFile    string
	sourceFile string
	tgzFiles   []string
	workers    int
	verbose    bool
}

func (a *arguments) parse() {

	// Assign variables
	flag.StringVar(&a.outFile, "out", "", "output filepath for .jsonl.gz dataset")
	flag.StringVar(&a.sourceFile, "sources", "", "csv file mapping archive names to sources names, otherwise use archive name")
	flag.IntVar(&a.workers, "parfactor", 1, "parallel processing factor")
	flag.BoolVar(&a.verbose, "verbose", false, "explain all skipped games to stderr")

	// Usage and parse
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gofilter [options] [-out outfile] [tgzfile1 tgzfile2 ... tgzfileN]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	a.tgzFiles = flag.Args()
}

func (a *arguments) check() error {
	if a.workers < 1 {
		return errors.New("parfactor must be at least 1")
	}
	if len(a.tgzFiles) == 0 {
		return errors.New("no tgz archives provided")
	}
	if a.outFile == "" {
		return errors.New("no output file provided")
	}
	if _, err := os.Stat(a.outFile); (err == nil) || os.IsExist(err) {
		fmt.Print("output file already exists, overwrite? (y/n) ")
		r := bufio.NewReader(os.Stdin)
		overwrite, _ := r.ReadString('\n')
		if strings.TrimSpace(overwrite) != "y" {
			return errors.New("did not overwrite file")
		}
	}

	return nil

}
