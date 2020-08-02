/*
Package weiqi implements Go game logic.

Rulesets available:

    New Zealand (default)    "NZ"   (situational superko, suicide allowed)
    American Go Association  "AGA"  (situational superko, suicide prohibited)
    Tromp-Taylor             "TT"   (positional superko, suicide allowed)
    unrestricted             ""     (no ko rule, suicide allowed)

Game scoring is not supported yet.*/
package weiqi

// Game stores Go game information and its methods allow for game control
type Game struct {
	turn       int8
	board      board
	prevTurns  []int8
	prevBoards []board
	prevHashes []int
	rules      rules
}

// NewGame starts a new game with unrestricted rules
func NewGame(height, width int) Game {
	b := newBoard(height, width)
	r, _ := newRules("")

	g := Game{turn: 1, board: b, rules: r}
	g.prevTurns = append(g.prevTurns, -1)
	g.prevBoards = append(g.prevBoards, g.board)
	g.prevHashes = append(g.prevHashes, 0)
	return g
}

// SetRules configures the ruleset used ("NZ", "AGA", "TT", or "")
func (g *Game) SetRules(ruleset string) error {
	r, err := newRules(ruleset)
	if err != nil {
		return err
	}
	g.rules = r
	return nil
}

func (g *Game) playOrCheck(m Move, playMode string) error {

	// Wrong player
	if m.Color != g.turn {
		return GameError{ErrWrongPlayer, m}
	}

	// Pass is always legal (if correct player)
	if m.pass {
		if playMode == "play" {
			g.turn = -m.Color
			g.prevTurns = append(g.prevTurns, m.Color)
			g.prevBoards = append(g.prevBoards, g.board)
			g.prevHashes = append(g.prevHashes, g.prevHashes[len(g.prevHashes)-1])
		}
		return nil
	}

	// Can never play a move outside of board
	if !g.board.exists(m.vertex) {
		return GameError{ErrOutsideBoard, m}
	}

	// Vertex not empty
	emptyColor := g.board.look(m.vertex)
	if emptyColor != 0 {
		return GameError{ErrVertexNotEmpty, m}
	}

	// Copy board and place move
	nextBoard := g.board.Copy()
	nextBoard.place(m)

	// Clear opponent stones
	oppStonesRemoved := false
	for _, adj := range m.vertex.adjacent(nextBoard.height, nextBoard.width) {
		adjColor := nextBoard.look(adj)
		if adjColor == -m.Color {
			group := newGroupIfDead(adj, nextBoard)
			if !group.alive {
				nextBoard.remove(group)
				oppStonesRemoved = true
			}
		}
	}

	// Clear own stones
	if !oppStonesRemoved {
		group := newGroupIfDead(m.vertex, nextBoard)
		if !group.alive {
			if g.rules.suicideForbidden {
				return GameError{ErrSuicide, m}
			}
			nextBoard.remove(group)
		}
	}

	// Check ko violation
	if g.rules.positionalSuperko || g.rules.situationalSuperko { // But only if ruleset deems it necessary
		for i := range g.prevTurns {
			if nextBoard.hash == g.prevHashes[i] { // First speedily compare hashes
				if nextBoard.Equals(g.prevBoards[i]) { // Then double check the real board
					if g.rules.positionalSuperko {
						return GameError{ErrPositionalSuperko, m}
					}
					if m.Color == g.prevTurns[i] {
						return GameError{ErrSituationalSuperko, m}
					}
				}
			}
		}
	}

	// Update game state
	if playMode == "play" {
		g.turn = -m.Color
		g.board = nextBoard
		g.prevTurns = append(g.prevTurns, m.Color)          // When modifying these,
		g.prevBoards = append(g.prevBoards, nextBoard)      // must also modify the
		g.prevHashes = append(g.prevHashes, nextBoard.hash) // early return cases above.
	}

	return nil

}

// Play plays a move if it is legal
func (g *Game) Play(m Move) error {
	return g.playOrCheck(m, "play")
}

// Check checks move legality but does not alter the game state
func (g *Game) Check(m Move) error {
	return g.playOrCheck(m, "check")
}

func (g Game) String() string {
	return g.board.String()
}
