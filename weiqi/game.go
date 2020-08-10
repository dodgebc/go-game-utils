/*
Package weiqi implements Go game logic.

Rulesets available:

    New Zealand (default)    "NZ"   (situational superko, suicide allowed)
    American Go Association  "AGA"  (situational superko, suicide prohibited)
    Tromp-Taylor             "TT"   (positional superko, suicide allowed)
    unrestricted             ""     (no ko rule, suicide allowed)

Game scoring is not supported yet.*/
package weiqi

import (
	"fmt"
)

// Game stores Go game information and its methods allow for game control
type Game struct {
	// game state
	turn  int8
	board board

	// game history
	prevMoves   []Move
	prevHashes  []int
	TrustHashes bool // rely only on hashes for ko

	// game rules
	SuicideForbidden   bool
	SituationalSuperko bool
	PositionalSuperko  bool

	// these eliminate new allocations on each turn
	workingGroup group
	nextBoard    board
}

// NewGame starts a new game with NZ rules as default
func NewGame(rows, cols int) Game {
	if (rows < 0) || (cols < 0) {
		panic("tried to set negative game size")
	}
	b := newBoard(rows, cols)
	b2 := newBoard(rows, cols)
	g := Game{turn: 1, board: b, nextBoard: b2}
	g.SetRules("NZ")
	return g
}

// SetRules configures the ruleset used ("NZ", "AGA", "TT", or "")
func (g *Game) SetRules(ruleset string) error {
	g.SuicideForbidden = false
	g.SituationalSuperko = false
	g.PositionalSuperko = false
	switch ruleset {
	case "NZ":
		g.SituationalSuperko = true
	case "AGA":
		g.SuicideForbidden = true
		g.SituationalSuperko = true
	case "TT":
		g.PositionalSuperko = true
	case "":
	default:
		return fmt.Errorf("did not recognize ruleset: %s", ruleset)
	}
	return nil
}

// Reset returns the game to its starting state
func (g *Game) Reset() {
	g.turn = 1
	g.board.clear()
	g.prevMoves = g.prevMoves[:0]
	g.prevHashes = g.prevHashes[:0]
}

// handles "play", "check", "setup" play modes
func (g *Game) playWithMode(m Move, playMode string) error {

	// Wrong player
	if m.Color != g.turn {
		if playMode != "setup" {
			return GameError{ErrWrongPlayer, m}
		}
	}

	// Pass is always legal (if correct player)
	if m.pass {
		if playMode != "check" {
			g.turn = -m.Color
			g.prevMoves = append(g.prevMoves, m)
			g.prevHashes = append(g.prevHashes, g.board.hash)
		}
		return nil
	}

	// Can never play a move outside of board
	if !g.board.exists(m.vertex) {
		return GameError{ErrOutsideBoard, m}
	}

	// Vertex not empty
	emptyColor := g.board.flatArray[m.vertex[0]*g.board.cols+m.vertex[1]]
	if emptyColor != 0 {
		if playMode != "setup" {
			return GameError{ErrVertexNotEmpty, m}
		}
	}

	// Place move
	g.nextBoard.CopyFrom(g.board)
	g.nextBoard.place(m)

	// Clear opponent stones
	oppStonesRemoved := false
	for i := 0; i < 2; i++ { // Loop over adjacent vertices
		for j := -1; j < 2; j += 2 {
			adj := vertex{m.vertex[0] + i*j, m.vertex[1] + (1-i)*j}
			if (adj[0] >= 0) && (adj[0] < g.nextBoard.rows) && (adj[1] >= 0) && (adj[1] < g.nextBoard.cols) {
				adjColor := g.nextBoard.flatArray[adj[0]*g.nextBoard.cols+adj[1]]
				if adjColor == -m.Color {
					g.workingGroup.expandAllIfDead(adj, g.nextBoard)
					if !g.workingGroup.alive {
						g.nextBoard.remove(g.workingGroup)
						oppStonesRemoved = true
					}
				}
			}
		}
	}

	// Clear own stones
	if !oppStonesRemoved {
		g.workingGroup.expandAllIfDead(m.vertex, g.nextBoard)
		if !g.workingGroup.alive {
			if g.SuicideForbidden && (playMode != "setup") {
				return GameError{ErrSuicide, m}
			}
			g.nextBoard.remove(g.workingGroup)
		}
	}

	// Check ko violation (if ruleset deems it necessary)
	if (g.PositionalSuperko || g.SituationalSuperko) && (playMode != "setup") {
		for i := range g.prevMoves {
			if g.nextBoard.hash == g.prevHashes[i] {
				confirmedRepeat := true
				if !g.TrustHashes {
					replayGame := NewGame(g.board.rows, g.board.cols) // Replay game to check boards
					for _, m := range g.prevMoves[:i+1] {
						replayGame.Setup(m)
					}
					if !g.nextBoard.Equals(replayGame.board) {
						confirmedRepeat = false
					}
				}
				if confirmedRepeat {
					if g.PositionalSuperko {
						return GameError{ErrPositionalSuperko, m}
					}
					if m.Color == g.prevMoves[i].Color {
						return GameError{ErrSituationalSuperko, m}
					}
				}
			}
		}
	}

	// Update game state (this is also potentially updated for passes above)
	if playMode != "check" {
		g.turn = -m.Color
		g.board.CopyFrom(g.nextBoard)
		g.prevMoves = append(g.prevMoves, m)
		g.prevHashes = append(g.prevHashes, g.board.hash)
	}
	return nil
}

// Play plays a move if it is legal
func (g *Game) Play(m Move) error {
	return g.playWithMode(m, "play")
}

// Check checks move legality but does not alter the game state
func (g *Game) Check(m Move) error {
	return g.playWithMode(m, "check")
}

// Setup allows arbitrary moves to be played for setup purposes.
// Everything behaves like a normal move except for legality checks.
// The only possible error is ErrOutsideBoard for invalid vertices.
func (g *Game) Setup(m Move) error {
	return g.playWithMode(m, "setup")
}

func (g Game) String() string {
	return g.board.String()
}

// CheckLegal conveniently checks if a game is legal and returns and error if arguments themselves are bad
func CheckLegal(rows, cols int, Setup, Moves []string, ruleset string) (bool, error) {
	if (rows < 0) || (cols < 0) {
		return false, fmt.Errorf("negative game size: %d %d", rows, cols)
	}
	g := NewGame(rows, cols)
	err := g.SetRules(ruleset)
	if err != nil {
		return false, err
	}
	for _, ms := range Setup {
		m, err := NewMoveFromString(ms)
		if err != nil {
			return false, err
		}
		err = g.Setup(m)
		if err != nil {
			return false, nil
		}
	}
	for _, ms := range Moves {
		m, err := NewMoveFromString(ms)
		if err != nil {
			return false, err
		}
		err = g.Play(m)
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}
