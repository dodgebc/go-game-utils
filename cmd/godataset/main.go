// godataset builds a .jsonl.gz dataset of games from a .tar.gz archive of SGF files
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/build/pargzip"
)

func main() {

	// Output file and source names
	var outFile, sourceFile string
	flag.StringVar(&outFile, "out", "", "output filepath for .jsonl.gz dataset")
	flag.StringVar(&sourceFile, "sources", "", "csv file with archive names and corresponding source names, otherwise use archive name")

	// Configuration for checking
	var deduplicate, checkLegal bool
	var ruleset string
	var minLength int
	var verbose bool
	flag.BoolVar(&deduplicate, "deduplicate", false, "remove games with duplicate move sequences")
	flag.BoolVar(&checkLegal, "checklegal", false, "check if games are legal under provided ruleset")
	flag.StringVar(&ruleset, "ruleset", "NZ", "ruleset to use for legality checking, \"NZ\", \"AGA\", \"TT\", or \"\"")
	flag.IntVar(&minLength, "minlength", 0, "minimum number of moves per game")
	flag.BoolVar(&verbose, "verbose", false, "explain all skipped games")

	// Parallelism
	var workers int
	flag.IntVar(&workers, "workers", 1, "number of concurrent workers to use")

	// Usage and input files
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: godataset [options] [tgzfile1 tgzfile2 ... tgzfileN]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	tgzFiles := flag.Args()

	// Load source names
	sourceNames := make(map[string]string)
	if sourceFile != "" {
		sfile, err := os.Open(sourceFile)
		if err != nil {
			log.Fatal(err)
		}
		sbytes, err := ioutil.ReadAll(sfile)
		if err != nil {
			log.Fatal(err)
		}
		lines := strings.Split(string(sbytes), "\n")
		for _, l := range lines {
			cols := strings.Split(l, ",")
			if len(cols) == 2 {
				sourceNames[strings.TrimSpace(cols[0])] = strings.TrimSpace(cols[1])
			}
		}
	}

	// Open .jsonl output file stream
	if outFile == "" {
		log.Fatal("no output path provided")
	}
	fout, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	// Compress output file stream
	gzipWriter := pargzip.NewWriter(fout)
	gzipWriter.Parallel = workers
	defer gzipWriter.Close()

	// Set up game checking
	if (ruleset != "NZ") && (ruleset != "AGA") && (ruleset != "TT") && (ruleset != "") {
		log.Fatalf("ruleset %q not recognized", ruleset)
	}
	checker := NewCheckManager(minLength, deduplicate, checkLegal, ruleset, verbose)

	// Loop over all archives
	for _, tgzFile := range tgzFiles {

		// Source name
		tgzName := filepath.Base(tgzFile)
		sourceName := tgzName
		if s, ok := sourceNames[tgzName]; ok {
			sourceName = s
		}
		progressUpdate := NewProgressUpdate(fmt.Sprintf("%s (%q)", tgzName, sourceName))

		// Create channels
		cancel := make(chan struct{})
		cerrLoader := make(chan error)
		cerrProcessor := make(chan error)
		cerrSaver := make(chan error)
		in := make(chan []byte, 1<<10)
		out := make(chan []byte, 1<<10)

		// Single loader and saver
		go func() {
			loader(cancel, cerrLoader, tgzFile, in, checker, progressUpdate)
			close(in)
		}()
		go func() {
			saver(cancel, cerrSaver, out, gzipWriter)
			cerrSaver <- nil
		}()

		// Many processors
		var wg sync.WaitGroup
		wg.Add(workers)
		for j := 0; j < workers; j++ {
			go func() {
				processor(cancel, cerrProcessor, in, out, checker, sourceName)
				wg.Done()
			}()
		}
		go func() {
			wg.Wait()
			close(out)
		}()

		// Wait for completion or an error
		select {
		case err := <-cerrLoader:
			close(cancel)
			log.Fatal(err)
		case err := <-cerrProcessor:
			close(cancel)
			log.Print(err)
		case err := <-cerrSaver:
			if err != nil {
				close(cancel)
				log.Fatal(err)
			}
		}

		// Final progress update and then zero checker
		progressUpdate.SetOther("failed", checker.NumFailed)
		if checker.MinLength > 0 {
			progressUpdate.SetOther("short", checker.NumShort)
		}
		if checker.CheckLegal {
			progressUpdate.SetOther("illegal", checker.NumIllegal)
		}
		if checker.Deduplicate {
			progressUpdate.SetOther("duplicate", checker.NumDuplicate)
		}
		progressUpdate.Close()
		checker.ZeroCounts()
	}

}
