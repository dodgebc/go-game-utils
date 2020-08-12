// gofilter filters a .jsonl.gz dataset of games with various options
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/dodgebc/go-utils/sgfgrab"
)

func main() {

	// Command line arguments
	var inFile, outFile, blacklistFile string
	var topCut, sample float64
	var minLength int
	var metaOnly bool //, anon, gameid bool
	flag.StringVar(&outFile, "out", "", "output filepath for filtered .jsonl.gz dataset")
	flag.StringVar(&blacklistFile, "blacklist", "", "filepath with list of regular expressions to exclude matching players, case-insensitive")
	flag.Float64Var(&topCut, "topcut", 0.0, "fraction of most frequent players to remove per source")
	flag.Float64Var(&sample, "sample", 1.0, "fraction of games to sample")
	flag.IntVar(&minLength, "minlength", 0, "discard games with fewer than this many moves")
	flag.BoolVar(&metaOnly, "metaonly", false, "strip move data to reduce size")
	//flag.BoolVar(&anon, "anon", false, "replace player name with unique player id")
	//flag.BoolVar(&gameid, "gameid", false, "add a unique game id to each game")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gofilter [options] [input filepath] \n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Check arguments
	if flag.NArg() != 1 {
		log.Fatal("expected exactly one input file")
	}
	if topCut < 0. || topCut > 1. {
		log.Fatal("topcut should be between 0 and 1")
	}
	if sample < 0. || sample > 1. {
		log.Fatal("sample should be between 0 and 1")
	}
	if outFile == "" {
		log.Fatal("no output path provided")
	}
	f, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	} else {
		f.Close()
	}
	inFile = flag.Arg(0)

	// Read blacklist
	var blacklist []*regexp.Regexp
	if blacklistFile != "" {
		blacklist = loadBlacklist(blacklistFile)
	}

	// Count player frequency
	sourcePlayerCounts := make(map[string]map[string]int)

	if topCut != 0.0 {
		pi := make(chan int)
		go progress("Counting player frequency", pi)
		i := 0

		in := make(chan sgfgrab.GameData)
		go readDataset(inFile, in, sample)

		for g := range in {

			// Update map with player counts
			if _, ok := sourcePlayerCounts[g.Source]; !ok {
				sourcePlayerCounts[g.Source] = make(map[string]int)
			}
			if len(g.BlackPlayer) > 0 {
				sourcePlayerCounts[g.Source][g.BlackPlayer]++
			}
			if len(g.WhitePlayer) > 0 {
				sourcePlayerCounts[g.Source][g.WhitePlayer]++
			}

			pi <- 1
			i++
			if i == 100000 {
				break
			}

		}
		close(pi)
	}

	// Build source blacklist based on frequency
	sourceBlacklist := make(map[string][]string)
	for source, playerCounts := range sourcePlayerCounts {
		sourceBlacklist[source] = computeTopCut(topCut, playerCounts)
	}

	//fmt.Println(sourceBlacklist)

	// Second pass to actually perform filtering
	pi := make(chan int)
	extra := newSafeCounter()
	go progressExtended("Filtering", pi, &extra)
	in := make(chan sgfgrab.GameData)
	out := make(chan sgfgrab.GameData)
	done := make(chan struct{})
	go readDataset(inFile, in, sample)
	go writeDataset(outFile, out, done)

	for g := range in {
		include := true

		// Check both blacklists
		for _, n := range sourceBlacklist[g.Source] {
			if (g.BlackPlayer == n) || (g.WhitePlayer == n) {
				include = false
				extra.Increment("topcut")
			}
		}
		for _, re := range blacklist {
			if re.MatchString(g.BlackPlayer) || re.MatchString(g.WhitePlayer) {
				include = false
				extra.Increment("blacklist")
			}
		}

		// Remove move data if requested
		if metaOnly {
			g.Moves = []string{}
			g.Setup = []string{}
		}

		// Filter by game length
		if g.Length < minLength {
			include = false
			extra.Increment("minlength")
		}

		// Send for writing
		if include {
			out <- g
		}

		pi <- 1
	}
	close(pi)
	close(out)
	<-done
}

func progress(description string, iterations <-chan int) {
	lastUpdate := time.Now()
	i := 0
	for it := range iterations {
		i += it
		if time.Since(lastUpdate).Seconds() > 0.5 {
			fmt.Printf("\r%s: %d", description, i)
			lastUpdate = time.Now()
		}
	}
	fmt.Println()
}

func progressExtended(description string, iterations <-chan int, extra *safeCounter) {
	lastUpdate := time.Now()
	i := 0
	for it := range iterations {
		i += it
		if time.Since(lastUpdate).Seconds() > 0.5 {
			fmt.Printf("\r%s: %d", description, i)
			for _, k := range extra.Keys() {
				fmt.Printf("\t%s: %d", k, extra.Get(k))
			}
			lastUpdate = time.Now()
		}
	}
	fmt.Println()
}

type safeCounter struct {
	m   map[string]int
	mux sync.Mutex
}

func newSafeCounter() safeCounter {
	return safeCounter{
		m: make(map[string]int),
	}
}

func (c *safeCounter) Set(s string, i int) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.m[s] = i
}

func (c *safeCounter) Increment(s string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.m[s]++
}

func (c *safeCounter) Get(s string) int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.m[s]
}

func (c *safeCounter) Keys() []string {
	c.mux.Lock()
	defer c.mux.Unlock()
	s := []string{}
	for k := range c.m {
		s = append(s, k)
	}
	return s
}
