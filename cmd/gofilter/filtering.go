package main

import (
	"bufio"
	"errors"
	"fmt"
	"hash/maphash"
	"log"
	"os"
	"regexp"
	"sort"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"github.com/dodgebc/go-game-utils/weiqi"
)

// filterFunc checks a game and return nil error if it should be accepted
type filterFunc func(g sgfgrab.GameData) error

// filterWrapper creates a filtered channel and a non-nil error channel using the given filterFunc
func filterWrapper(in <-chan sgfgrab.GameData, filter filterFunc, workers int) (<-chan sgfgrab.GameData, <-chan error) {
	out := make(chan sgfgrab.GameData)
	cerr := make(chan error)

	go func() {
		defer close(out)
		defer close(cerr)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for g := range in {
					if err := filter(g); err != nil {
						cerr <- fmt.Errorf("game %d: %s", g.GameID, err)
					} else {
						out <- g
					}
				}
			}()
		}

	}()
	return out, cerr
}

// Filter games which are too short
func filterMinLength(in <-chan sgfgrab.GameData, minLength int, workers int) (<-chan sgfgrab.GameData, <-chan error) {
	filter := func(g sgfgrab.GameData) error {
		if g.Length < minLength {
			return fmt.Errorf("game length %d too short", g.Length)
		}
		return nil
	}
	return filterWrapper(in, filter, workers)
}

// Filter games where a player matches the blacklist
func filterBlacklist(in <-chan sgfgrab.GameData, blacklistFile string, workers int) (<-chan sgfgrab.GameData, <-chan error) {

	// Load the blacklist
	var blacklistRe []*regexp.Regexp
	f, err := os.Open(blacklistFile)
	if err != nil {
		log.Fatalf("failed to open blacklist file: %s", err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) > 0 {
			re, err := regexp.Compile("(?i)" + scanner.Text())
			if err != nil {
				log.Fatalf("failed to compile blacklist regexp: %s", err)
			}
			blacklistRe = append(blacklistRe, re)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("blacklist file read error: %s", err)
	}

	// Apply filter
	filter := func(g sgfgrab.GameData) error {
		for _, re := range blacklistRe {
			if re.MatchString(g.BlackPlayer) {
				return fmt.Errorf("player %q matched blacklist regexp %q", g.BlackPlayer, re)
			} else if re.MatchString(g.WhitePlayer) {
				return fmt.Errorf("player %q matched blacklist regexp %q", g.WhitePlayer, re)
			}
		}
		return nil
	}
	return filterWrapper(in, filter, workers)
}

// Filter games with very frequent players (likely bots)
func filterTopCut(in <-chan sgfgrab.GameData, topCut float64, sourcePlayerCounts map[string]map[string]int, workers int) (<-chan sgfgrab.GameData, <-chan error) {

	// Compute source blacklist
	sourceBlacklist := make(map[string][]string)
	for source, playerCounts := range sourcePlayerCounts {
		var counts []int
		var names []string
		for _, c := range playerCounts {
			counts = append(counts, c)
		}
		sort.Ints(counts)
		cutoff := counts[int(float64(len(counts)-1)*(1-topCut))]
		for n, c := range playerCounts {
			if c > cutoff {
				names = append(names, n)
			}
		}
		sourceBlacklist[source] = names
	}

	// Apply filter
	filter := func(g sgfgrab.GameData) error {
		for _, name := range sourceBlacklist[g.Source] {
			if (name == g.BlackPlayer) || (name == g.WhitePlayer) {
				return fmt.Errorf("player %q appeared too frequenty", name)
			}
		}
		return nil
	}
	return filterWrapper(in, filter, workers)
}

// Filter duplicate games
func filterDuplicate(in <-chan sgfgrab.GameData, workers int) (<-chan sgfgrab.GameData, <-chan error) {

	// Initialize hashing
	hashTable := make(map[uint64]bool)
	var mux sync.Mutex
	var hash maphash.Hash

	// Apply filter
	filter := func(g sgfgrab.GameData) error {
		mux.Lock()
		defer mux.Unlock()
		hash.Reset()
		for _, m := range g.Moves {
			hash.WriteString(m)
		}
		hash.WriteString(g.Winner)
		sum := hash.Sum64()
		if hashTable[sum] {
			return errors.New("duplicate game")
		}
		hashTable[sum] = true
		return nil
	}
	return filterWrapper(in, filter, workers)
}

// Filter illegal games
func filterIllegal(in <-chan sgfgrab.GameData, ruleset string, workers int) (<-chan sgfgrab.GameData, <-chan error) {
	filter := func(g sgfgrab.GameData) error {
		return weiqi.CheckLegal(g.Size[0], g.Size[1], g.Setup, g.Moves, ruleset)
	}
	return filterWrapper(in, filter, workers)
}
