// godataset builds a .jsonl.gz dataset of games from a .tar.gz archive of SGF files
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dodgebc/handy-go/progress"
)

func main() {
	log.SetFlags(0)

	// Command line argumetns
	var args arguments
	args.parse()
	if err := args.check(); err != nil {
		log.Fatal(err)
	}

	// Load source names
	sourceNames := make(map[string]string)
	if args.sourceFile != "" {
		f, err := os.Open(args.sourceFile)
		if err != nil {
			log.Fatalf("failed to open sources file: %s", err)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			cols := strings.Split(scanner.Text(), ",")
			if len(cols) == 2 {
				sourceNames[strings.TrimSpace(cols[0])] = strings.TrimSpace(cols[1])
			} else if len(cols) != 1 {
				log.Fatalf("sources file should have two columns, found %d", len(cols))
			}
		}
	}

	// Prepare to write dataset
	out := make(chan []byte, 1<<6)
	done := saver(out, args.outFile)
	defer func() { <-done }()
	defer close(out)

	// Loop over all archives
	for _, tgzFile := range args.tgzFiles {
		cancel := make(chan struct{})

		// Source name
		tgzName := filepath.Base(tgzFile)
		sourceName := tgzName
		if s, ok := sourceNames[tgzName]; ok {
			sourceName = s
		}
		mon := progress.NewMonitor(fmt.Sprintf("%s (%q)", tgzName, sourceName))

		// Load and count files
		in := make(chan []byte)
		go func() {
			defer close(in)
			temp, cerr := loader(tgzFile, cancel)
			go func() {
				if err := <-cerr; err != nil {
					log.Println(err)
					close(cancel)
				}
			}()
			for b := range temp {
				in <- b
				mon.Increment(1)
			}
		}()

		// Parse SGF data and log errors if requested
		thisOut, cerr := parser(in, sourceName, args.workers, cancel)
		go func() {
			for err := range cerr {
				if args.verbose {
					log.Println(err)
				}
				mon.IncrementCounter("failed", 1)
			}
		}()

		// Save data
		for b := range thisOut {
			out <- b
		}
		mon.Close()
		break
	}
}
