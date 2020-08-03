package weiqi

import "fmt"

type vertex [2]int

// Move stores the color and intersection of a move
type Move struct {
	Color  int8
	vertex vertex
	pass   bool
}

// NewMove creates a Move with coordinates
func NewMove(color int8, row, col int) Move {
	return Move{Color: color, vertex: vertex{row, col}}
}

// NewMovePass creates a pass Move
func NewMovePass(color int8) Move {
	return Move{Color: color, pass: true}
}

// NewMoveFromString parses an SGF-style string like "Bcd" (pass is "W", not "Wtt").
// The maximum coordinate in this format is Z (52), use NewMove for larger sizes.
func NewMoveFromString(moveString string) (Move, error) {

	// Check length of string
	pass := false
	switch len(moveString) {
	case 1:
		pass = true
	case 3:
	default:
		return Move{}, fmt.Errorf("invalid move string: %q", moveString)
	}

	// Parse color
	var color int8
	switch moveString[0] {
	case 'B':
		color = 1
	case 'W':
		color = -1
	default:
		return Move{}, fmt.Errorf("invalid color in move: %q", moveString)
	}

	// Parse vertex
	if pass {
		return Move{Color: color, pass: true}, nil
	}
	row, err1 := letterToCoordinate(moveString[2])
	col, err2 := letterToCoordinate(moveString[1])
	if (err1 != nil) || (err2 != nil) {
		return Move{}, fmt.Errorf("invalid coordinates in move: %q", moveString)
	}
	return Move{Color: color, vertex: vertex{row, col}}, nil
}

func (m Move) String() string {
	playerLetter := "?"
	switch m.Color {
	case 1:
		playerLetter = "B"
	case -1:
		playerLetter = "W"
	}
	rowLetter, err1 := coordinateToLetter(m.vertex[0])
	colLetter, err2 := coordinateToLetter(m.vertex[1])
	if err1 != nil {
		rowLetter = "?"
	}
	if err2 != nil {
		colLetter = "?"
	}
	return playerLetter + colLetter + rowLetter
}

func letterToCoordinate(letter byte) (int, error) {
	value := int(letter)
	if (value >= 97) && (value <= 122) { // lowercase
		return value - 97, nil
	}
	if (value >= 65) && (value <= 90) { // uppercase
		return value - 65 + 26, nil
	}
	return 0, fmt.Errorf("invalid letter: %q", letter)
}

func coordinateToLetter(coordinate int) (string, error) {
	if (coordinate < 0) || (coordinate >= 52) {
		return "", fmt.Errorf("cannot convert coordinate to string: %d", coordinate)
	}
	if coordinate < 26 { // lowercase
		return string(byte(coordinate + 97)), nil
	}
	return string(byte(coordinate - 26 + 65)), nil // uppercase
}
