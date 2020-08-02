package weiqi

import "fmt"

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

// NewMoveFromString parses an SGF-style string like "Bcd" (pass is "W", not "Wtt")
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

// Equals compares two moves
func (m Move) Equals(m2 Move) bool {
	if m.Color != m2.Color {
		return false
	}
	if m.vertex != m2.vertex {
		return false
	}
	return true
}

func (m Move) String() string {
	s := "?"
	switch m.Color {
	case 1:
		s = "B"
	case -1:
		s = "W"
	}
	return s + m.vertex.String()
}

// vertex stores coordinates of an intersection
// potentially replace with [2]int for performance
type vertex struct {
	row, col int
}

// adjacent returns adjacent vertices (PERFORMANCE MATTERS HERE)
// possibly remove this function, too much allocation overhead
// behavior may be unintuitive if vertex is outside the board
func (v vertex) adjacent(height, width int) []vertex {
	adj := make([]vertex, 0, 4) // main cost is this allocation, shouldn't really need to do it this way
	if v.row-1 >= 0 {
		adj = append(adj, vertex{v.row - 1, v.col})
	}
	if v.row+1 < height {
		adj = append(adj, vertex{v.row + 1, v.col})
	}
	if v.col-1 >= 0 {
		adj = append(adj, vertex{v.row, v.col - 1})
	}
	if v.col+1 < width {
		adj = append(adj, vertex{v.row, v.col + 1})
	}
	return adj
}

func (v vertex) Equals(v2 vertex) bool {
	return (v.row == v2.row) && (v.col == v2.col)
}

func (v vertex) String() string {
	rowLetter, err1 := coordinateToLetter(v.row)
	colLetter, err2 := coordinateToLetter(v.col)
	if err1 != nil {
		rowLetter = "?"
	}
	if err2 != nil {
		colLetter = "?"
	}
	return colLetter + rowLetter
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
		return "", fmt.Errorf("invalid coordinate: %d", coordinate)
	}
	if coordinate < 26 { // lowercase
		return string(byte(coordinate + 97)), nil
	}
	return string(byte(coordinate - 26 + 65)), nil // uppercase
}
