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

	// Input/output
	outFile    string
	sourceFile string
	tgzFiles   []string

	// Filters
	gameid      bool
	minLength   int
	deduplicate bool
	checkLegal  bool
	ruleset     string

	// Execution
	workers int
	verbose bool
}

func (a *arguments) parse() {

	// Assign variables
	flag.StringVar(&a.outFile, "out", "", "output filepath for .jsonl.gz dataset")
	flag.StringVar(&a.sourceFile, "sources", "", "csv file mapping archive names to sources names, otherwise use archive name")
	flag.BoolVar(&a.gameid, "gameid", false, "add a unique ID to each game")
	flag.IntVar(&a.minLength, "minlength", 0, "minimum number of moves per game")
	flag.BoolVar(&a.deduplicate, "deduplicate", false, "remove games with duplicate move sequences")
	flag.BoolVar(&a.checkLegal, "checklegal", false, "check if games are legal under provided ruleset")
	flag.StringVar(&a.ruleset, "ruleset", "", "ruleset to use for legality checking: \"NZ\", \"AGA\", \"TT\", or \"\"")
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
	if a.minLength < 0 {
		return errors.New("minlength must be non-negative")
	}
	switch a.ruleset {
	case "NZ", "TT", "AGA", "":
	default:
		return fmt.Errorf("ruleset %q not supported", a.ruleset)
	}
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
