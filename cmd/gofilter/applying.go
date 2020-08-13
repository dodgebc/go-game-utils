package main

import (
	"log"
	"math/rand"

	"github.com/dodgebc/go-game-utils/sgfgrab"
)

// GENERATING IDs SHOULD BE DONE BY A SINGLE GOROUTINE THROUGHOUT EXECUTION

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

// Strip moves and setup from game
func applyMetaOnly(in <-chan sgfgrab.GameData) <-chan sgfgrab.GameData {
	apply := func(g *sgfgrab.GameData) {
		g.Moves = g.Moves[:0]
		g.Setup = g.Setup[:0]
	}
	return applyWrapper(in, apply)
}

// Generate a unique game ID and add it to the game (ONLY ONE SHOULD BE RUNNING)
/*func applyGameID(in <-chan sgfgrab.GameData) <-chan sgfgrab.GameData {

	// Keep track of used game IDs
	GameIDUsed := make(map[uint32]bool)

	// Use a unique random number
	apply := func(g *sgfgrab.GameData) {
		tryID := rand.Uint32()
		i := 0
		for _, ok := GameIDUsed[tryID]; ok; _, ok = GameIDUsed[tryID] {
			tryID = rand.Uint32()
			if i++; i > 50 {
				log.Fatal("random number cycle detected")
			}
		}
		GameIDUsed[tryID] = true
		g.GameID = tryID
	}
	return applyWrapper(in, apply)
}*/

// Generate a unique ID for each player to replace their names (ONLY ONE SHOULD BE RUNNING)
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
	return applyWrapper(in, apply)
}
