// gofilter filters a .jsonl.gz dataset of games with various options
package main

import (
	"log"
	"os"
	"runtime/pprof"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"github.com/dodgebc/handy-go/progress"
)

func main() {
	log.SetFlags(0)
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer f.Close() // error handling omitted for example
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	// Filtering configuration
	var args arguments
	args.parse()
	if err := args.check(); err != nil {
		log.Fatal(err)
	}

	// Count player frequency first if required
	sourcePlayerCounts := make(map[string]map[string]int)
	if args.topCut != 0.0 {
		games := unmarshalGame(readGzipLines(args.inFile, args.sample), args.workers)
		mon := progress.NewMonitor("counting player frequency")
		for g := range games {
			if _, ok := sourcePlayerCounts[g.Source]; !ok {
				sourcePlayerCounts[g.Source] = make(map[string]int)
			}
			if len(g.BlackPlayer) > 0 {
				sourcePlayerCounts[g.Source][g.BlackPlayer]++
			}
			if len(g.WhitePlayer) > 0 {
				sourcePlayerCounts[g.Source][g.WhitePlayer]++
			}
			mon.Increment(1)
		}
		mon.Close()
	}

	// Start progress monitor
	mon := progress.NewMonitor("filtering")
	defer mon.Close()
	mon.StartCounter("accepted")
	if args.blacklistFile != "" {
		mon.StartCounter("blacklist")
	}
	if args.topCut != 0.0 {
		mon.StartCounter("topcut")
	}
	if args.deduplicate {
		mon.StartCounter("duplicate")
	}
	if args.checkLegal {
		mon.StartCounter("illegal")
	}

	// Load and count games
	games := make(chan sgfgrab.GameData)
	go func() {
		defer close(games)
		temp := unmarshalGame(readGzipLines(args.inFile, args.sample), args.workers)
		for g := range temp {
			games <- g
			mon.Increment(1)
		}
	}()

	// Collect games and errors
	out := make(chan sgfgrab.GameData)
	var wg sync.WaitGroup
	wg.Add(args.workers) // This is for the game collectors
	go func() {
		wg.Wait()
		close(out)
	}()
	gameCollect := func(in <-chan sgfgrab.GameData) {
		for g := range in {
			out <- g
			mon.IncrementCounter("accepted", 1)
		}
		wg.Done()
	}
	errCollect := func(in <-chan error, name string) {
		wg.Add(1)
		for err := range in {
			if args.verbose {
				log.Println(err)
			}
			mon.IncrementCounter(name, 1)
		}
		wg.Done()
	}

	// Filtering pipeline
	for i := 0; i < args.workers; i++ {
		needle := (<-chan sgfgrab.GameData)(games)
		var errChan <-chan error
		if args.minLength > 0 {
			needle, errChan = filterMinLength(needle, args.minLength)
			go errCollect(errChan, "short")
		}
		if args.blacklistFile != "" {
			needle, errChan = filterBlacklist(needle, args.blacklistFile)
			go errCollect(errChan, "blacklist")
		}
		if args.topCut != 0.0 {
			needle, errChan = filterTopCut(needle, args.topCut, sourcePlayerCounts)
			go errCollect(errChan, "topcut")
		}
		if args.deduplicate {
			needle, errChan = filterDuplicate(needle)
			go errCollect(errChan, "duplicate")
		}
		if args.checkLegal {
			needle, errChan = filterIllegal(needle, args.ruleset)
			go errCollect(errChan, "illegal")
		}
		if args.gameID {
			needle = applyGameID(needle)
		}
		if args.playerID {
			needle = applyPlayerID(needle)
		}
		if args.metaOnly {
			needle = applyMetaOnly(needle)
		}
		go gameCollect(needle)
	}

	// Write games
	done := writeGzipLines(marshalGame(out, args.workers), args.outFile)
	<-done
}
