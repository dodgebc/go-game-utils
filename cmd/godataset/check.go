package main

import (
	"errors"
	"hash/maphash"
	"sync"

	"github.com/dodgebc/go-utils/sgfgrab"
	"github.com/dodgebc/go-utils/weiqi"
)

// CheckManager handles deduplication and legality checking
type CheckManager struct {

	// configuration
	MinLength   int
	Deduplicate bool
	CheckLegal  bool
	Ruleset     string
	Verbose     bool

	// counters
	NumFailed    int
	NumDuplicate int
	NumIllegal   int
	NumShort     int

	// hashing for duplicates
	hashTable map[uint64]bool
	seed      maphash.Seed
	mux       sync.Mutex
}

// NewCheckManager properly initializes a CheckManager
func NewCheckManager(minLength int, deduplicate, checkLegal bool, ruleset string, verbose bool) *CheckManager {
	checker := &CheckManager{
		MinLength:   minLength,
		Deduplicate: deduplicate,
		CheckLegal:  checkLegal,
		Ruleset:     ruleset,
		Verbose:     verbose,
	}
	if deduplicate {
		checker.hashTable = make(map[uint64]bool)
		checker.seed = maphash.MakeSeed()
	}
	return checker
}

// Check evaluates a game and returns whether it should be included (nil means yes)
func (checker *CheckManager) Check(g sgfgrab.GameData) error {
	checker.mux.Lock()
	defer checker.mux.Unlock()
	if len(g.Moves) < checker.MinLength {
		checker.NumShort++
		return errors.New("too short")
	}
	if checker.Deduplicate {
		sum := checker.computeMovesHash(g.Moves)
		if checker.hashTable[sum] {
			checker.NumDuplicate++
			return errors.New("duplicate")
		}
		checker.hashTable[sum] = true
	}
	if checker.CheckLegal {
		err := weiqi.CheckLegal(g.Size[0], g.Size[1], g.Setup, g.Moves, checker.Ruleset)
		if err != nil {
			checker.NumIllegal++
			return err
		}
	}
	return nil
}

// AddFailed records a failed game parse
func (checker *CheckManager) AddFailed(n int) {
	checker.mux.Lock()
	defer checker.mux.Unlock()
	checker.NumFailed += n
}

// ZeroCounts clears counts without destroying duplicates hash
func (checker *CheckManager) ZeroCounts() {
	checker.NumFailed = 0
	checker.NumDuplicate = 0
	checker.NumIllegal = 0
	checker.NumShort = 0
}

// computeMovesHash computes a hash for the move sequence
func (checker *CheckManager) computeMovesHash(moves []string) uint64 {
	var hash maphash.Hash
	hash.SetSeed(checker.seed)
	for _, m := range moves {
		hash.WriteString(m)
	}
	return hash.Sum64()
}
