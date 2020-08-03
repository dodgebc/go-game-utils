package weiqi

import (
	"errors"
	"math/rand"
	"testing"
)

func BenchmarkAlphaGo(b *testing.B) {
	moveSequence := []string{
		"Bpd", "Wdp", "Bcd", "Wqp", "Bop", "Woq", "Bnq", "Wpq", "Bcn", "Wfq",
		"Bmp", "Wpo", "Biq", "Wec", "Bhd", "Wcg", "Bed", "Wcj", "Bdc", "Wbp",
		"Bnc", "Wqi", "Bep", "Weo", "Bdk", "Wfp", "Bck", "Wdj", "Bej", "Wei",
		"Bfi", "Weh", "Bfh", "Wbj", "Bfk", "Wfg", "Bgg", "Wff", "Bgf", "Wmc",
		"Bmd", "Wlc", "Bnb", "Wid", "Bhc", "Wjg", "Bpj", "Wpi", "Boj", "Woi",
		"Bni", "Wnh", "Bmh", "Wng", "Bmg", "Wmi", "Bnj", "Wmf", "Bli", "Wne",
		"Bnd", "Wmj", "Blf", "Wmk", "Bme", "Wnf", "Blh", "Wqj", "Bkk", "Wik",
		"Bji", "Wgh", "Bhj", "Wge", "Bhe", "Wfd", "Bfc", "Wki", "Bjj", "Wlj",
		"Bkh", "Wjh", "Bml", "Wnk", "Bol", "Wok", "Bpk", "Wpl", "Bqk", "Wnl",
		"Bkj", "Wii", "Brk", "Wom", "Bpg", "Wql", "Bcp", "Wco", "Boe", "Wrl",
		"Bsk", "Wrj", "Bhg", "Wij", "Bkm", "Wgi", "Bfj", "Wjl", "Bkl", "Wgl",
		"Bfl", "Wgm", "Bch", "Wee", "Beb", "Wbg", "Bdg", "Weg", "Ben", "Wfo",
		"Bdf", "Wdh", "Bim", "Whk", "Bbn", "Wif", "Bgd", "Wfe", "Bhf", "Wih",
		"Bbh", "Wci", "Bho", "Wgo", "Bor", "Wrg", "Bdn", "Wcq", "Bpr", "Wqr",
		"Brf", "Wqg", "Bqf", "Wjc", "Bgr", "Wsf", "Bse", "Wsg", "Brd", "Wbl",
		"Bbk", "Wak", "Bcl", "Whn", "Bin", "Whp", "Bfr", "Wer", "Bes", "Wds",
		"Bah", "Wai", "Bkd", "Wie", "Bkc", "Wkb", "Bgk", "Wib", "Bqh", "Wrh",
		"Bqs", "Wrs", "Boh", "Wsl", "Bof", "Wsj", "Bni", "Wnj", "Boo", "Wjp",
	}
	g := NewGame(19, 19)
	_ = g.SetRules("NZ")
	for i := 0; i < b.N; i++ {
		g.Reset()
		for _, m := range moveSequence {
			move, err := NewMoveFromString(m)
			if err != nil {
				b.Fatalf("benchmark contained bad move string: %s", m)
			}
			err = g.Play(move)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkRandomGame(b *testing.B) {
	g := NewGame(19, 19)
	g.SetRules("NZ")
	for i := 0; i < b.N; i++ {
		g.Reset()
		rand.Seed(1)
		for j := 0; j < 300; j++ {
			var player int8
			if j%2 == 0 {
				player = 1
			} else {
				player = -1
			}
			row := rand.Intn(g.board.height)
			col := rand.Intn(g.board.width)
			g.Play(NewMove(player, row, col))
		}
	}
}

func TestBasicGamePlayCheck(t *testing.T) {

	g := NewGame(3, 3)

	// This sequence of moves tests a simple case of each possible game error
	// "m" is the move string
	// "e" is the expected error
	// "r" is the ruleset to be applied BEFORE the move
	testTable := []map[string]interface{}{
		{"m": "Bba"},
		{"m": "Bcc", "e": ErrWrongPlayer},
		{"m": "W"},
		{"m": "Bee", "e": ErrOutsideBoard},
		{"m": "Bba", "e": ErrVertexNotEmpty},
		{"m": "Bab"},
		{"m": "Waa", "e": ErrSuicide, "r": "AGA"},
		{"m": "Waa", "r": "NZ"},
		{"m": "B"},
		{"m": "Wbb"},
		{"m": "B"},
		{"m": "Wca"},
		{"m": "B"},
		{"m": "Waa"},
		{"m": "Bba", "e": ErrSituationalSuperko},
		{"m": "Bba", "e": ErrPositionalSuperko, "r": "TT"},
		{"m": "Bba", "r": ""},
		{"m": "W"},
		{"m": "Bcb"},
		{"m": "Wbc", "r": "TT"},
		{"m": "Bac"},
		{"m": "Waa"},
		{"m": "Bab", "e": ErrPositionalSuperko},
		{"m": "Bab", "r": "NZ"},
	}

	var err1, err2 error

	for i := range testTable {

		// Switch up the rules at various points
		if testTable[i]["r"] != nil {
			g.SetRules(testTable[i]["r"].(string))
		}

		// Play and Check should be consistent
		m, err := NewMoveFromString(testTable[i]["m"].(string))
		if err != nil {
			t.Fatalf("test contained bad move string: %s", testTable[i]["m"])
		}
		err1 = g.Check(m)
		err2 = g.Play(m)
		if err1 != err2 {
			t.Fatalf("inconsistent error from Check and Play on move %d: %v, %v", i, err1, err2)
		}

		// Verify that we get the error we expect
		var expected error
		if testTable[i]["e"] != nil {
			expected = testTable[i]["e"].(error)
		}
		if !errors.Is(err1, expected) {
			t.Fatalf("unexpected game error on move %d: %v", i, err1)
		}
	}

}

func TestGameSetup(t *testing.T) {
	g := NewGame(3, 3)
	g.Play(NewMove(1, 0, 1))
	err := g.Setup(NewMove(-1, 5, 5))
	if !errors.Is(err, ErrOutsideBoard) {
		t.Fatal("setup allowed outside of board move")
	}
	err = g.Setup(NewMove(-1, 0, 1))
	if err != nil {
		t.Fatal("setup did not allow an illegal move")
	}
	if g.turn != -1 {
		t.Fatal("setup switched player turn")
	}
	if g.board.flatArray[1] != -1 {
		t.Fatal("setup did not place a stone")
	}
	if g.prevBoards[0].flatArray[1] != 1 {
		t.Fatal("setup corrupted prevBoards")
	}
	if g.prevHashes[0] == g.board.hash {
		t.Fatal("setup did not update hash")
	}
}

func TestGameReset(t *testing.T) {
	g := NewGame(19, 19)
	startBoard := g.board.Copy()
	g.Play(NewMove(1, 1, 1))
	g.Reset()
	if !g.board.Equals(startBoard) {
		t.Fatal("reset game did not match start board")
	}
	if !(g.turn == 1) {
		t.Fatal("reset game had wrong start player")
	}
	if !(len(g.prevTurns) == 0 && len(g.prevBoards) == 0 && len(g.prevHashes) == 0) {
		t.Fatal("reset game retained previous information")
	}
}

func TestBasicGroupExpansion(t *testing.T) {

	// Setup (see diagram)
	b := newBoard(3, 3)
	b.place(NewMove(1, 0, 0)) // X X .
	b.place(NewMove(1, 0, 1)) // . X .
	b.place(NewMove(1, 1, 1)) // X O X
	b.place(NewMove(1, 2, 0))
	b.place(NewMove(-1, 2, 1))
	b.place(NewMove(1, 2, 2))

	p := newGroup(vertex{0, 1}, b)
	if len(p.interior) != 3 {
		t.Fatal("group expansion failed to find all connected vertices")
	}
	if len(p.edge) != 0 {
		t.Fatal("group expansion left vertices in edge")
	}
	if !p.alive {
		t.Fatal("group expansion incorrectly marked group as dead")
	}

	p = newGroup(vertex{2, 1}, b)
	if p.alive {
		t.Fatal("group expansion incorrectly marked group as alive")
	}

}

func TestBasicGroupRemoval(t *testing.T) {

	b := newBoard(2, 2)

	// Diagonal stones
	b.place(Move{Color: 1, vertex: vertex{0, 0}})
	b.place(Move{Color: 1, vertex: vertex{1, 1}})
	b.remove(newGroup(vertex{0, 0}, b))
	if b.look(vertex{1, 1}) != 1 {
		t.Fatal("group removal took diagonal stones")
	}

	// Adjacent stones
	b.place(Move{Color: 1, vertex: vertex{1, 0}})
	b.remove(newGroup(vertex{1, 0}, b))
	if b.look(vertex{1, 1}) != 0 {
		t.Fatal("group removal did not take adjacent stones")
	}

}

func TestBoardComparison(t *testing.T) {

	// Create boards and check basic equality
	b1 := newBoard(5, 5)
	b2 := b1.Copy()
	if !b1.Equals(b2) {
		t.Fatal("empty boards not equal")
	}

	// Make sure copy was deep
	b1.place(NewMove(1, 1, 1))
	if b2.look(vertex{1, 1}) != 0 {
		t.Fatal("board copy was not deep")
	}
	b2.place(NewMove(1, 1, 1))

	if b1.hash != b2.hash {
		t.Fatal("board copy had different hash")
	}

}

func TestBoardManipulation(t *testing.T) {

	b := newBoard(5, 6)

	// Exist
	v := vertex{4, 6}
	if b.exists(v) {
		t.Fatalf("vertex (%d, %d) existed on %dx%d board", v[0], v[1], b.height, b.width)
	}
	v = vertex{4, 5}
	if !b.exists(v) {
		t.Fatalf("vertex (%d, %d) did not exist on %dx%d board", v[0], v[1], b.height, b.width)
	}

	// Place and look
	m := Move{Color: 1, vertex: v}
	b.place(m)
	if b.look(v) != 1 {
		t.Fatalf("board place and look failed")
	}

}

func TestMoveParse(t *testing.T) {

	// Basic move parse
	m, err := NewMoveFromString("Wab")
	if err != nil {
		t.Fatal(err)
	}
	if m != NewMove(-1, 1, 0) {
		t.Fatalf("move \"Wab\" parsed incorrectly as %q", m)
	}

	// Pass move parse
	m, err = NewMoveFromString("B")
	if err != nil {
		t.Fatal(err)
	}
	if m != NewMovePass(1) {
		t.Fatalf("move \"B\" parsed incorrectly as %q", m)
	}

	// Large move parse
	m, err = NewMoveFromString("BAZ")
	if err != nil {
		t.Fatal(err)
	}
	if m != NewMove(1, 51, 26) {
		t.Fatalf("move \"BAZ\" parsed incorrectly as %q", m)
	}

	// Malformed moves
	m, err = NewMoveFromString("Zab")
	if err == nil {
		t.Fatalf("parsed invalid move \"Zab\" as %q", m)
	}
	m, err = NewMoveFromString("W~5")
	if err == nil {
		t.Fatalf("parsed invalid move \"W~5\" as %q", m)
	}

}

func TestMovePrint(t *testing.T) {

	// Large move
	m := NewMove(1, 5, 26)
	if m.String() != "BAf" {
		t.Fatalf("move \"BAf\" printed incorrectly as %q", m)
	}

	// Invalid move
	m = NewMove(-1, 5, 100)
	if m.String() != "W?f" {
		t.Fatalf("invalid move \"Bf?\" printed incorrectly as %q", m)
	}
}
