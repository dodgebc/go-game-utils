package main

import (
	"bufio"
	"errors"
	"fmt"
	"hash/maphash"
	"log"
	"math/rand"
	"os"
	"regexp"
	"sort"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"github.com/dodgebc/go-game-utils/weiqi"
)

// applyFunc modifies a game
type applyFunc func(g *sgfgrab.GameData)

// applyWrapper creates a channel with modifications using the given applyFunc
func applyWrapper(in <-chan sgfgrab.GameData, apply applyFunc) <-chan sgfgrab.GameData {
	out := make(chan sgfgrab.GameData)
	go func() {
		defer close(out)
		for g := range in {
			apply(&g)
			out <- g
		}
	}()
	return out
}

// filterFunc checks a game and return nil error if it should be accepted
type filterFunc func(g sgfgrab.GameData) error

// filterWrapper creates a filtered channel and a non-nil error channel using the given filterFunc
func filterWrapper(in <-chan sgfgrab.GameData, filter filterFunc) (<-chan sgfgrab.GameData, <-chan error) {
	out := make(chan sgfgrab.GameData)
	cerr := make(chan error)
	go func() {
		defer close(out)
		defer close(cerr)
		for g := range in {
			if err := filter(g); err != nil {
				cerr <- err
			} else {
				out <- g
			}
		}
	}()
	return out, cerr
}

// Strip moves and setup from game
func applyMetaOnly(in <-chan sgfgrab.GameData) <-chan sgfgrab.GameData {
	apply := func(g *sgfgrab.GameData) {
		g.Moves = g.Moves[:0]
		g.Setup = g.Setup[:0]
	}
	return applyWrapper(in, apply)
}

// Generate a unique game ID and add it to the game
func applyGameID(in <-chan sgfgrab.GameData) <-chan sgfgrab.GameData {

	// Keep track of used game IDs
	GameIDUsed := make(map[uint32]bool)

	// Use a unique random number
	apply := func(g *sgfgrab.GameData) {
		tryID := rand.Uint32()
		for _, ok := GameIDUsed[tryID]; ok; {
			tryID = rand.Uint32()
		}
		GameIDUsed[tryID] = true
		g.GameID = tryID
	}
	return applyWrapper(in, apply)
}

// Generate a unique ID for each player to replace their names
func applyPlayerID(in <-chan sgfgrab.GameData) <-chan sgfgrab.GameData {

	// Initialize "two way" lookup table
	SourcePlayerIDTable := make(map[string]map[string]uint32)
	PlayerIDUsed := make(map[uint32]bool)

	// If player has not been seen before, use a unique random number
	apply := func(g *sgfgrab.GameData) {
		if _, ok := SourcePlayerIDTable[g.Source]; !ok {
			SourcePlayerIDTable[g.Source] = make(map[string]uint32)
		}
		for i := 0; i < 2; i++ {
			var name string
			if i == 0 {
				name = g.BlackPlayer
			} else {
				name = g.WhitePlayer
			}
			var tryID uint32
			if name != "" { // Empty names get a zero player ID
				if ID, ok := SourcePlayerIDTable[g.Source][name]; ok {
					tryID = ID
				} else {
					tryID = rand.Uint32()
					for _, ok := PlayerIDUsed[tryID]; ok; {
						tryID = rand.Uint32()
					}
					SourcePlayerIDTable[g.Source][name] = tryID
					PlayerIDUsed[tryID] = true
				}
			}
			if i == 0 {
				g.BlackID = tryID
			} else {
				g.WhiteID = tryID
			}
		}
	}
	return applyWrapper(in, apply)
}

// Filter games which are too short
func filterMinLength(in <-chan sgfgrab.GameData, minLength int) (<-chan sgfgrab.GameData, <-chan error) {
	filter := func(g sgfgrab.GameData) error {
		if g.Length < minLength {
			return fmt.Errorf("game length %d too short", g.Length)
		}
		return nil
	}
	return filterWrapper(in, filter)
}

// Filter games where a player matches the blacklist
func filterBlacklist(in <-chan sgfgrab.GameData, blacklistFile string) (<-chan sgfgrab.GameData, <-chan error) {

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
	return filterWrapper(in, filter)
}

// Filter games with very frequent players (likely bots)
func filterTopCut(in <-chan sgfgrab.GameData, topCut float64, sourcePlayerCounts map[string]map[string]int) (<-chan sgfgrab.GameData, <-chan error) {

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
	return filterWrapper(in, filter)
}

// Filter duplicate games
func filterDuplicate(in <-chan sgfgrab.GameData) (<-chan sgfgrab.GameData, <-chan error) {

	// Initialize hashing
	hashTable := make(map[uint64]bool)
	var hash maphash.Hash

	// Apply filter
	filter := func(g sgfgrab.GameData) error {
		hash.Reset()
		for _, m := range g.Moves {
			hash.WriteString(m)
		}
		hash.WriteString(g.Winner)
		sum := hash.Sum64()
		if hashTable[sum] {
			return errors.New("duplicate game")
		}
		return nil
	}
	return filterWrapper(in, filter)
}

// Filter illegal games
func filterIllegal(in <-chan sgfgrab.GameData, ruleset string) (<-chan sgfgrab.GameData, <-chan error) {
	filter := func(g sgfgrab.GameData) error {
		return weiqi.CheckLegal(g.Size[0], g.Size[1], g.Setup, g.Moves, ruleset)
	}
	return filterWrapper(in, filter)
}
