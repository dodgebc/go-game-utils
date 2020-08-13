// godataset builds a .jsonl.gz dataset of games from a .tar.gz archive of SGF files
package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
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
	out := make(chan []byte, 4096*args.workers)
	done := saver(out, args.outFile)
	defer func() { <-done }()
	defer close(out)

	// Prepare game ID source
	gameIDSource := generateGameID(done)

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
		in := make(chan []byte, 4096*args.workers)
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
		thisOut, cerr := parser(in, sourceName, gameIDSource, args.workers, cancel)
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
	}
}

// Source from which to generate unique game IDs throughout program execution
func generateGameID(done <-chan struct{}) <-chan uint32 {
	out := make(chan uint32)
	go func() {
		defer close(out)

		// Keep track of used game IDs
		GameIDUsed := make(map[uint32]bool)

		// Use a unique random number
		for {
			tryID := rand.Uint32()
			i := 0
			for _, ok := GameIDUsed[tryID]; ok; _, ok = GameIDUsed[tryID] {
				tryID = rand.Uint32()
				if i++; i > 50 {
					log.Fatal("random number cycle detected")
				}
			}
			GameIDUsed[tryID] = true
			select {
			case out <- tryID:
			case <-done:
				return
			}
		}
	}()
	return out
}
