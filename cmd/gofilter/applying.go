package main

import (
	"log"
	"math/rand"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
)

// applyFunc modifies a game
type applyFunc func(g *sgfgrab.GameData)

// applyWrapper creates a channel with modifications using the given applyFunc
func applyWrapper(in <-chan sgfgrab.GameData, apply applyFunc, workers int) <-chan sgfgrab.GameData {
	out := make(chan sgfgrab.GameData)

	go func() {
		defer close(out)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for g := range in {
					apply(&g)
					out <- g
				}
			}()
		}
	}()

	return out
}

// Strip moves and setup from game
func applyMetaOnly(in <-chan sgfgrab.GameData, workers int) <-chan sgfgrab.GameData {
	apply := func(g *sgfgrab.GameData) {
		g.Moves = g.Moves[:0]
		g.Setup = g.Setup[:0]
	}
	return applyWrapper(in, apply, workers)
}

// Generate a unique ID for each player to replace their names
func applyPlayerID(in <-chan sgfgrab.GameData, workers int) <-chan sgfgrab.GameData {

	// Initialize "two way" lookup table
	SourcePlayerIDTable := make(map[string]map[string]uint32)
	PlayerIDUsed := make(map[uint32]bool)
	var mux sync.Mutex

	// If player has not been seen before, use a unique random number
	apply := func(g *sgfgrab.GameData) {
		mux.Lock()
		defer mux.Unlock()
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
					i := 0
					for _, ok := PlayerIDUsed[tryID]; ok; _, ok = PlayerIDUsed[tryID] {
						tryID = rand.Uint32()
						if i++; i > 50 {
							log.Fatal("random number cycle detected")
						}
					}
					SourcePlayerIDTable[g.Source][name] = tryID
					PlayerIDUsed[tryID] = true
				}
			}
			if i == 0 {
				g.BlackID = tryID
				g.BlackPlayer = ""
			} else {
				g.WhiteID = tryID
				g.WhitePlayer = ""
			}
		}
	}
	return applyWrapper(in, apply, workers)
}
