package main

import (
	"bufio"
	"errors"
	"fmt"
	"hash/maphash"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/dodgebc/go-game-utils/weiqi"
)

// filterFunc checks a game and return nil error if it should be accepted
type filterFunc func(p packet) error

// filterWrapper creates a filtered channel and a non-nil error channel using the given filterFunc
func filterWrapper(in <-chan packet, filter filterFunc, workers int) (<-chan packet, <-chan packet) {
	good := make(chan packet)
	bad := make(chan packet)

	go func() {
		defer close(good)
		defer close(bad)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for p := range in {
					p.err = filter(p)
					if p.err != nil {
						bad <- p
					} else {
						good <- p
					}
				}
			}()
		}

	}()
	return good, bad
}

// Filter games which are too short
func filterMinLength(in <-chan packet, minLength int, workers int) (<-chan packet, <-chan packet) {
	filter := func(p packet) error {
		if p.game.Length < minLength {
			return fmt.Errorf("game length %d too short", p.game.Length)
		}
		return nil
	}
	return filterWrapper(in, filter, workers)
}

// Filter duplicate games
func filterDuplicate(in <-chan packet, workers int) (<-chan packet, <-chan packet) {

	// Initialize hashing
	hashTable := make(map[uint64]bool)
	var mux sync.Mutex
	var hash maphash.Hash

	// Apply filter
	filter := func(p packet) error {
		mux.Lock()
		defer mux.Unlock()
		hash.Reset()
		for _, m := range p.game.Moves {
			hash.WriteString(m)
		}
		hash.WriteString(p.game.Winner)
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
func filterIllegal(in <-chan packet, ruleset string, workers int) (<-chan packet, <-chan packet) {
	filter := func(p packet) error {
		return weiqi.CheckLegal(p.game.Size[0], p.game.Size[1], p.game.Setup, p.game.Moves, ruleset)
	}
	return filterWrapper(in, filter, workers)
}

// applyFunc modifies a game
type applyFunc func(p *packet)

// applyWrapper creates a channel with modifications using the given applyFunc
func applyWrapper(in <-chan packet, apply applyFunc, workers int) <-chan packet {
	out := make(chan packet)

	go func() {
		defer close(out)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for p := range in {
					apply(&p)
					out <- p
				}
			}()
		}
	}()

	return out
}

// Add a unique game id
func applyGameID(in <-chan packet, workers int) <-chan packet {

	// Keep track of used game IDs
	GameIDUsed := make(map[uint32]bool)
	var mux sync.Mutex
	src := rand.NewSource(1)
	random := rand.New(src)

	// Use a unique random number
	apply := func(p *packet) {
		mux.Lock()
		defer mux.Unlock()
		tryID := random.Uint32()
		i := 0
		for used := GameIDUsed[tryID]; used; used = GameIDUsed[tryID] {
			tryID = random.Uint32()
			if i++; i > 100 {
				log.Fatal("random number cycle detected")
			}
		}
		GameIDUsed[tryID] = true
		p.game.GameID = tryID
	}
	return applyWrapper(in, apply, workers)
}

// Add source name (expects a channel on which to receive the current tgzName)
func applySourceName(in <-chan packet, sourceFile string, workers int) <-chan packet {

	// Load source names
	sourceNames := make(map[string]string)
	f, err := os.Open(sourceFile)
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
	f.Close()

	apply := func(p *packet) {
		if sourceName, ok := sourceNames[p.tgzName]; ok {
			p.game.Source = sourceName
		} else {
			p.game.Source = p.tgzName
		}
	}
	return applyWrapper(in, apply, workers)
}

// Strip move data
func applyMetaOnly(in <-chan packet, workers int) <-chan packet {
	apply := func(p *packet) {
		p.game.Moves = p.game.Moves[:0]
		p.game.Setup = p.game.Setup[:0]
	}
	return applyWrapper(in, apply, workers)
}
