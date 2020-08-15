// godataset builds a .jsonl.gz dataset of games from a .tar.gz archive of SGF files
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"github.com/dodgebc/handy-go/progress"
)

type packet struct {
	game    sgfgrab.GameData
	err     error
	tgzName string
}

func main() {
	log.SetFlags(0)

	// Command line argumetns
	var args arguments
	args.parse()
	if err := args.check(); err != nil {
		log.Fatal(err)
	}

	// Collect for monitoring
	monChan := make(chan string)
	collect := func(ps <-chan packet, kind string) {
		for p := range ps {
			if p.err != nil && args.verbose {
				if p.game.GameID != 0 {
					log.Printf("game %d: %s\n", p.game.GameID, p.err)
				} else {
					log.Println(p.err)
				}
			}
			monChan <- kind
		}
	}

	// Pipeline
	in := make(chan packet, 4096*args.workers)
	good := (<-chan packet)(in)
	bad := make(<-chan packet)
	if args.gameid {
		good = applyGameID(good, args.workers)
	}
	if args.sourceFile != "" {
		good = applySourceName(good, args.sourceFile, args.workers)
	}
	if args.minLength != 0 {
		good, bad = filterMinLength(good, args.minLength, args.workers)
		go collect(bad, "short")
	}
	if args.deduplicate {
		good, bad = filterDuplicate(good, args.workers)
		go collect(bad, "duplicate")
	}
	if args.checkLegal {
		good, bad = filterIllegal(good, args.ruleset, args.workers)
		go collect(bad, "illegal")
	}
	out := make(chan packet, 4096*args.workers)
	go func() {
		for p := range good {
			out <- p
			monChan <- "accepted"
		}
		close(out)
	}()
	finishedAll := writeGzipLines(marshalGame(out, args.workers), args.outFile)
	defer func() { <-finishedAll }()

	// Loop over all archives
	for _, tgzFile := range args.tgzFiles {
		tgzName := filepath.Base(tgzFile)

		// Monitor progress and un-count
		var total sync.WaitGroup
		finish := make(chan struct{})
		finished := make(chan struct{})
		mon := progress.NewMonitor(fmt.Sprintf("%s", tgzName))
		mon.StartCounter("malformed")
		if args.minLength != 0 {
			mon.StartCounter("short")
		}
		if args.deduplicate {
			mon.StartCounter("duplicate")
		}
		if args.checkLegal {
			mon.StartCounter("illegal")
		}
		mon.StartCounter("accepted")
		go func() {
			for {
				select {
				case kind := <-monChan:
					mon.IncrementCounter(kind, 1)
					total.Add(-1)
				case <-finish:
					mon.Close()
					close(finished)
					return
				}
			}
		}()

		// Load from archive
		sgfBytes := make(chan []byte)
		go func() {
			for b := range readTgzSgf(tgzFile) {
				mon.Increment(1)
				sgfBytes <- b
			}
			close(sgfBytes)
		}()

		// Send into pipeline and count
		packets := parseGame(sgfBytes, tgzName, args.workers)
		for p := range packets {
			total.Add(1)
			if p.err == nil {
				in <- p
			} else {
				monChan <- "malformed"
			}
		}
		total.Wait()
		close(finish)
		<-finished
	}
	close(in)
}
