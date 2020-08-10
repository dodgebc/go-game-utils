package weiqi

import (
	"math/rand"
	"strings"
)

// maximum precomputed hash table size
const preMaxSize = 19

// This avoid expensive hash table re-initialization
// for games of a certain size or less.
var preHashTable []int

func init() {
	preHashTable = make([]int, preMaxSize*preMaxSize*2)
	for i := range preHashTable {
		preHashTable[i] = rand.Int()
	}
}

// board holds the current Go board
type board struct {
	rows, cols int
	flatArray  []int8
	hash       int
	hashTable  []int // Zobrist hashing
}

func newBoard(rows, cols int) board {
	b := board{rows: rows, cols: cols}
	b.flatArray = make([]int8, rows*cols)
	if rows*cols <= preMaxSize*preMaxSize {
		b.hashTable = preHashTable
	} else {
		// Board size too large, need to rebuild hash table
		// If many games will be large, recommend increasing preMaxSize
		rand.Seed(1)
		b.hashTable = make([]int, rows*cols*2)
		for i := range b.hashTable {
			b.hashTable[i] = rand.Int()
		}
	}
	return b
}

// exists checks if the vertex is on the board
func (b *board) exists(v vertex) bool {
	if (v[0] >= 0) && (v[0] < b.rows) && (v[1] >= 0) && (v[1] < b.cols) {
		return true
	}
	return false
}

// look retrieves the color at a vertex
// not using this except for tests & stringer because it does not get inlined
func (b *board) look(v vertex) int8 {
	return b.flatArray[v[0]*b.cols+v[1]]
}

// place places a move and updates the board hash
func (b *board) place(m Move) {
	if !m.pass {
		i := m.vertex[0]*b.cols + m.vertex[1]
		b.flatArray[i] = m.Color
		b.hash = b.hash ^ b.hashTable[i*2+int(b.flatArray[i]+1)/2]
	}
}

// remove removes a group and updates the board hash
func (b *board) remove(p group) {
	for _, v := range p.interior {
		i := v[0]*b.cols + v[1]
		b.hash = b.hash ^ b.hashTable[i*2+int(b.flatArray[i]+1)/2]
		b.flatArray[i] = 0
	}
}

// clear removes all stones and resets hash
func (b *board) clear() {
	for i := range b.flatArray {
		b.flatArray[i] = 0
	}
	b.hash = 0
}

func (b *board) Equals(b2 board) bool {
	if (b.rows != b2.rows) || (b.cols != b2.cols) {
		return false
	}
	for i := range b.flatArray {
		if b.flatArray[i] != b2.flatArray[i] {
			return false
		}
	}
	return true
}

func (b *board) Copy() board {
	b2 := board{rows: b.rows, cols: b.cols}
	b2.flatArray = make([]int8, b2.rows*b2.cols)
	copy(b2.flatArray, b.flatArray)
	b2.hash = b.hash
	b2.hashTable = b.hashTable
	return b2
}

func (b *board) CopyFrom(b2 board) {
	if (b.rows*b.cols != b2.rows) || (b.cols != b2.cols) {
		b.flatArray = make([]int8, b2.rows*b2.cols)
		b.rows = b2.rows
		b.cols = b2.cols
	}
	copy(b.flatArray, b2.flatArray)
	b.hash = b2.hash
	b.hashTable = b2.hashTable
}

func (b board) String() string {
	var rowCrosses, colCrosses []int // Where to put crosses (update this for other board sizes)
	if (b.rows == 19) && (b.cols == 19) {
		rowCrosses = []int{3, 3, 3, 9, 9, 9, 15, 15, 15}
		colCrosses = []int{3, 9, 15, 3, 9, 15, 3, 9, 15}
	}

	stringRows := make([]string, b.rows+1)
	stringRows[0] = "  "
	for i := 0; i < b.rows; i++ {
		stringRows[i+1] = string(rune(i+97)) + " "
		for j := 0; j < b.cols; j++ {
			if i == 0 {
				stringRows[0] += string(rune(j+97)) + " "
			}
			switch b.look(vertex{i, j}) {
			case 1:
				stringRows[i+1] += "X "
			case -1:
				stringRows[i+1] += "O "
			default:
				isCross := false
				for k := range rowCrosses {
					if (i == rowCrosses[k]) && (j == colCrosses[k]) {
						isCross = true
					}
				}
				if isCross {
					stringRows[i+1] += "+ "
				} else {
					stringRows[i+1] += ". "
				}
			}
		}
	}
	return strings.Join(stringRows, "\n") + "\n"
}

// Untested, loads board from string like one produced by String() below
// Possibly useful for providing desired output in tests
// NOT UPDATED WITH FLAT ARRAY
/*func newBoardFromString(s string) (board, error) {

	var array [][]int8

	// Split into rows by lines
	rows := strings.Split(s, "\n")
	for i, row := range rows {
		if i == 0 { // Column labels
			continue
		}
		array = append(array, make([]int8, 0))

		// Split into columns by spaces
		points := strings.Split(row, " ")
		for j, x := range points {
			if j == 0 { // Row labels
				continue
			}

			// Parse value
			if len(x) == 0 {
				continue
			}
			switch x[0] {
			case 'X':
				array[i-1] = append(array[i-1], 1)
			case 'O':
				array[i-1] = append(array[i-1], -1)
			case '.', '+':
				array[i-1] = append(array[i-1], 0)
			}
		}
	}

	// Check result
	rows := len(array)
	if (rows < 1) || (rows > 26) {
		return board{}, fmt.Errorf("parsed board has unexpected rows: %v", rows)
	}
	cols := len(array[0])
	for i := range array {
		if len(array[i]) != cols {
			return board{}, fmt.Errorf("parsed board rows not fixed length: %v %v", cols, len(array[i]))
		}
	}

	// Construct board
	board := newBoard(rows, cols)
	board.array = array
	return board, nil

}*/
