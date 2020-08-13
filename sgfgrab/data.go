package sgfgrab

import (
	"errors"
	"fmt"
)

// ErrAlreadyExists means that a property was already recorded for the game
var ErrAlreadyExists = errors.New("property already exists")

// GameData stores important parsed game data
type GameData struct {

	// must manually set these fields if desired
	Source  string `json:",omitempty"`
	GameID  uint32 `json:",omitempty"`
	BlackID uint32 `json:",omitempty"`
	WhiteID uint32 `json:",omitempty"`

	// critical fields where zero means something
	Size     [2]int  // (rows, cols) >= 1
	Komi     float64 //
	Handicap int     // >=0, unlike SGF
	Length   int     // number of actual game moves

	// optional fields where zero means nothing
	Winner      string   `json:",omitempty"` // "B", "W", or "" (no winner)
	Score       float64  `json:",omitempty"` //
	End         string   `json:",omitempty"` // "Scored", "Time", "Resign", "Forfeit", or ""
	BlackRank   string   `json:",omitempty"` // [0-9]{1,2}[kdp]
	WhiteRank   string   `json:",omitempty"` // [0-9]{1,2}[kdp]
	BlackPlayer string   `json:",omitempty"` //
	WhitePlayer string   `json:",omitempty"` //
	Time        int      `json:",omitempty"` // >=0, seconds
	Year        int      `json:",omitempty"` //
	Setup       []string `json:",omitempty"` // matches ([BW][a-z]{2})?
	Moves       []string `json:",omitempty"` // matches ([BW][a-z]{2})?

	alreadyRecorded [11]bool
}

// Finalize checks for any inconsistencies and fills in defaults
func (g *GameData) Finalize() error {

	// Size default
	if !g.alreadyRecorded[0] {
		g.Size = [2]int{19, 19}
	}

	// No setup stones despite handicap (maybe they are recorded as game moves)
	if (len(g.Setup) == 0) && (g.Handicap != 0) {
		if len(g.Moves) >= g.Handicap {
			g.Setup = g.Moves[:g.Handicap]
			g.Moves = g.Moves[g.Handicap:]
		}
	}

	// No handicap despite setup stones (try setting it to number of setup stones)
	if (len(g.Setup) != 0) && (g.Handicap == 0) {
		g.Handicap = len(g.Setup)
	}

	// Check number of handicap stones
	if len(g.Setup) != g.Handicap {
		return fmt.Errorf("handicap %d has %d setup stones", g.Handicap, len(g.Setup))
	}

	// Check color of handicap stones (should be all black)
	for i := range g.Setup {
		if g.Setup[i][:1] != "B" {
			return fmt.Errorf("setup stone not black")
		}
	}

	// Replace "tt" with pass where applicable
	if g.Size[0]*g.Size[1] <= 19*19 {
		for i := range g.Setup {
			if g.Setup[i][1:] == "tt" {
				g.Setup[i] = g.Setup[i][:1]
			}
		}
		for i := range g.Moves {
			if g.Moves[i][1:] == "tt" {
				g.Moves[i] = g.Moves[i][:1]
			}
		}
	}

	// Record game length
	g.Length = len(g.Moves)

	return nil
}

// AddProperty (possibly) parses an identifier/value pair
func (g *GameData) AddProperty(identifier, value string) error {
	switch identifier {
	case "SZ":
		if g.alreadyRecorded[0] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseSize(value)
		if err != nil {
			return err
		}
		g.Size = v
		g.alreadyRecorded[0] = true
		return nil
	case "KM":
		if g.alreadyRecorded[1] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseKomi(value)
		if err != nil {
			return err
		}
		g.Komi = v
		g.alreadyRecorded[1] = true
		return nil
	case "HA":
		if g.alreadyRecorded[2] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseHandicap(value)
		if err != nil {
			return err
		}
		g.Handicap = v
		g.alreadyRecorded[2] = true
		return nil
	case "RE":
		if g.alreadyRecorded[3] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v1, v2, v3, err := ParseResult(value)
		if err != nil {
			return err
		}
		g.Winner = v1
		g.Score = v2
		g.End = v3
		g.alreadyRecorded[3] = true
		return nil
	case "BR":
		if g.alreadyRecorded[4] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseRank("B", value)
		if err != nil {
			return err
		}
		g.BlackRank = v
		g.alreadyRecorded[4] = true
		return nil
	case "WR":
		if g.alreadyRecorded[5] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseRank("W", value)
		if err != nil {
			return err
		}
		g.WhiteRank = v
		g.alreadyRecorded[5] = true
		return nil
	case "PB":
		if g.alreadyRecorded[6] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		g.BlackPlayer = value
		g.alreadyRecorded[6] = true
		return nil
	case "PW":
		if g.alreadyRecorded[7] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		g.WhitePlayer = value
		g.alreadyRecorded[7] = true
		return nil
	case "TM":
		if g.alreadyRecorded[8] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseTime(value)
		if err != nil {
			return err
		}
		g.Time = v
		g.alreadyRecorded[8] = true
		return nil
	case "DT":
		if g.alreadyRecorded[10] {
			return fmt.Errorf("%w: %s %s", ErrAlreadyExists, identifier, value)
		}
		v, err := ParseDate(value)
		if err != nil {
			return err
		}
		g.Year = v
		g.alreadyRecorded[10] = true
		return nil
	case "B", "W", "AB", "AW":
		player := identifier[len(identifier)-1:]
		v, err := ParseMove(player, value)
		if err != nil {
			return err
		}
		if identifier[:1] == "A" {
			g.Setup = append(g.Setup, v)
		} else {
			g.Moves = append(g.Moves, v)
		}
		return nil
	}

	return nil
}

// Equals compares two games
func (g *GameData) Equals(g2 GameData) bool {
	switch {
	case g.Size != g2.Size:
		return false
	case g.Komi != g2.Komi:
		return false
	case g.Handicap != g2.Handicap:
		return false
	case g.Winner != g2.Winner:
		return false
	case g.Score != g2.Score:
		return false
	case g.End != g2.End:
		return false
	case g.BlackRank != g2.BlackRank:
		return false
	case g.WhiteRank != g2.WhiteRank:
		return false
	case g.BlackPlayer != g2.BlackPlayer:
		return false
	case g.WhitePlayer != g2.WhitePlayer:
		return false
	case g.Time != g2.Time:
		return false
	case g.Year != g2.Year:
		return false
	}
	if len(g.Moves) != len(g2.Moves) {
		return false
	}
	for i := range g.Moves {
		if g.Moves[i] != g2.Moves[i] {
			return false
		}
	}
	if len(g.Setup) != len(g2.Setup) {
		return false
	}
	for i := range g.Setup {
		if g.Setup[i] != g2.Setup[i] {
			return false
		}
	}
	return true
}
