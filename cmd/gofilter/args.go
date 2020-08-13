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
	inFile  string
	outFile string
	sample  float64

	// Filters
	minLength     int
	blacklistFile string
	topCut        float64
	deduplicate   bool
	checkLegal    bool
	ruleset       string

	// Apply modifications
	metaOnly bool
	playerID bool

	// Execution
	verbose bool
	workers int
}

func (a *arguments) parse() {

	// Assign variables
	flag.StringVar(&a.inFile, "in", "", "input filepath for .jsonl.gz dataset")
	flag.StringVar(&a.outFile, "out", "", "output filepath for filtered .jsonl.gz dataset")
	flag.Float64Var(&a.sample, "sample", 1.0, "fraction of games to sample")

	flag.IntVar(&a.minLength, "minlength", 0, "minimum number of moves per game")
	flag.StringVar(&a.blacklistFile, "blacklist", "", "filepath with case-insensitive regular expressions to exclude players")
	flag.Float64Var(&a.topCut, "topcut", 0.0, "fraction of most frequent players to remove per source (requires an additional scan)")
	flag.BoolVar(&a.deduplicate, "deduplicate", false, "remove games with duplicate move sequences")
	flag.BoolVar(&a.checkLegal, "checklegal", false, "check if games are legal under provided ruleset")
	flag.StringVar(&a.ruleset, "ruleset", "", "ruleset to use for legality checking: \"NZ\", \"AGA\", \"TT\", or \"\"")

	flag.BoolVar(&a.metaOnly, "metaonly", false, "strip move data to reduce size")
	flag.BoolVar(&a.playerID, "anon", false, "replace player name with unique player id")

	flag.BoolVar(&a.verbose, "verbose", false, "explain all skipped games to stderr")
	flag.IntVar(&a.workers, "parfactor", 1, "parallel processing factor")

	// Usage and parse
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gofilter [options] [-in infile] [-out outfile]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
}

func (a *arguments) check() error {

	// Check filters
	if a.topCut < 0. || a.topCut > 1. {
		return errors.New("topcut must be between 0 and 1")
	}
	if a.sample < 0. || a.sample > 1. {
		return errors.New("sample must be between 0 and 1")
	}
	if a.workers < 1 {
		return errors.New("parfactor must be at least 1")
	}
	if a.minLength < 0 {
		return errors.New("minlength must be non-negative")
	}
	switch a.ruleset {
	case "NZ", "TT", "AGA", "":
	default:
		return fmt.Errorf("ruleset %q not supported", a.ruleset)
	}

	// Check input and output files
	if a.inFile == "" {
		return errors.New("no input file provided")
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
	f, err := os.Create(a.outFile)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("could not create output file: %s", err)
	}
	return nil
}
