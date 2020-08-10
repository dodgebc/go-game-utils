package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/build/pargzip"
)

func main() {

	// Output file
	var outFile string
	flag.StringVar(&outFile, "out", "dataset.jsonl.gz", "output path")

	// Checking configuration
	var deduplicate, checkLegal bool
	var minLength int
	var verbose bool
	flag.BoolVar(&deduplicate, "deduplicate", false, "remove games with duplicate move sequences")
	flag.BoolVar(&checkLegal, "checklegal", false, "check if games are legal under NZ rules")
	flag.IntVar(&minLength, "minlength", 5, "minimum number of moves per game")
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

	// Open .jsonl output file stream
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
	checker := NewCheckManager(minLength, deduplicate, checkLegal, verbose)

	// Loop over all archives
	for _, tgzFile := range tgzFiles {

		// Create channels
		cancel := make(chan struct{})
		cerrLoader := make(chan error)
		cerrProcessor := make(chan error)
		cerrSaver := make(chan error)
		in := make(chan []byte)
		out := make(chan []byte)

		// Single loader and saver
		go func() {
			loader(cancel, cerrLoader, tgzFile, in, checker)
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
				processor(cancel, cerrProcessor, in, out, checker)
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

		checker.ZeroCounts()
	}

}
