/*
Package sgf provides simple SGF parsing capabilities.*/
package sgf

/*
From the SGF spec:

Collection = GameTree { GameTree }
GameTree   = "(" Sequence { GameTree } ")"
Sequence   = Node { Node }
Node       = ";" { Property }
Property   = PropIdent PropValue { PropValue }
PropIdent  = UcLetter { UcLetter }
PropValue  = "[" CValueType "]"
CValueType = (ValueType | Compose)
ValueType  = (None | Number | Real | Double | Color | SimpleText | Text | Point  | Move | Stone)
*/

import (
	"fmt"
	"strings"
	"unicode"
)

// Collection is a sequence of games
type Collection []*GameTree

// NewCollection parses text in SGF format into a Collection
func NewCollection(sgfText string) (Collection, error) {

	// Organization:
	// Collection is treated as just another GameTree until the end
	// Splitting trees and marking values is done in this function
	// Parsing into node/property structure is done in addNodeRune

	if sgfText[0] != '(' {
		return Collection{}, fmt.Errorf("collection did not start with open parenthesis")
	}

	// Parse setup
	brackOpen := false
	escaped := false
	justClosed := false
	justOpened := false
	var root GameTree // Treat the collection as just another GameTree
	var stack []*GameTree
	stack = append(stack, &root)

	// Iterative tree parsing (single pass)
	for _, r := range sgfText {

		// Determine when inside or outside bracket
		isValue := false
		if brackOpen {
			isValue = true
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == ']' {
				brackOpen = false
				isValue = false
			}
		} else if r == '[' {
			brackOpen = true
		} else if r == ']' {
			return Collection{}, fmt.Errorf("missing open bracket")
		}

		// Check for bad tree, i.e. (ab(c)d) is an error
		if justClosed && !unicode.IsSpace(r) {
			if (r != ')') && (r != '(') {
				return Collection{}, fmt.Errorf("bad tree")
			}
			justClosed = false
		}

		// Check for bad node, i.e. (;a;b(c)) is an error, so is (;a;b((;c))) or (;a())
		if justOpened && !unicode.IsSpace(r) {
			if r != ';' {
				return Collection{}, fmt.Errorf("bad sequence")
			}
			justOpened = false
		}

		// Expand tree
		if isValue {
			err := stack[len(stack)-1].addNodeRune(r, true)
			if err != nil {
				return Collection{}, err
			}
		} else {
			if r == '(' {
				add := new(GameTree)
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, add)
				stack = append(stack, add)
				justOpened = true
			} else if r == ')' {
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
					justClosed = true
				} else {
					return Collection{}, fmt.Errorf("missing open parenthesis")
				}
			} else {
				err := stack[len(stack)-1].addNodeRune(r, false)
				if err != nil {
					return Collection{}, err
				}
			}
		}
	}
	if brackOpen {
		return Collection{}, fmt.Errorf("missing close bracket")
	}
	if len(stack) > 1 {
		return Collection{}, fmt.Errorf("missing close parenthesis")
	}
	if len(root.Children) == 0 {
		return Collection{}, fmt.Errorf("collection contained no games")
	}
	for i := range root.Children {
		if len(root.Children[i].Nodes) == 0 {
			panic("game tree had no nodes")
		}
		root.Children[i].IsRoot = true
	}
	return root.Children, nil
}

// Equals checks Collection equality recursively
func (c Collection) Equals(c2 Collection) bool {
	if len(c) != len(c2) {
		return false
	}
	for i := range c {
		if !c[i].Equals(c2[i]) {
			return false
		}
	}
	return true
}

func (c Collection) String() string {
	s := ""
	for i := range c {
		s += c[i].String()
	}
	return s
}

// GameTree is a sequence of nodes potentially followed by other GameTrees (per SGF spec)
type GameTree struct {
	Nodes    []Node
	Children []*GameTree
	IsRoot   bool

	// state variables for parsing
	currentIdentifier strings.Builder
	currentValue      strings.Builder
	newIdentifier     bool
	escaped           bool
}

// Node stores a sequence of properties
type Node []Property

// Property stores an identifier and its values
type Property struct {
	Identifier string
	Values     []string
}

func (gt *GameTree) addNodeRune(r rune, isValue bool) error {

	if isValue { // Inside a value
		if r == '\r' { // Skip carriage returns for simplicity
			return nil
		}
		if gt.escaped {
			gt.escaped = false
			if r == '\n' { // Soft line break
				return nil
			}
		} else if r == '\\' { // Don't record escape backslashes
			gt.escaped = true
			return nil
		}
		if unicode.IsSpace(r) {
			r = ' ' // Convert whitespace to a space
		}
		gt.currentValue.WriteRune(r)

	} else { // Control rune
		if gt.escaped {
			// Should not be able to occur because of parsing in NewGameTree
			panic("escaped control character")
		}

		if r == ';' { // Start of a new node
			if gt.currentValue.Len() != 0 {
				return fmt.Errorf("value did not end: %s", gt.currentValue.String())
			}
			if gt.newIdentifier {
				return fmt.Errorf("identifier did not end: %s", gt.currentIdentifier.String())
			}
			gt.currentIdentifier.Reset()
			gt.Nodes = append(gt.Nodes, Node{})

		} else if r == '[' { // Start of a new value
			i := len(gt.Nodes) - 1
			ident := gt.currentIdentifier.String()
			if i < 0 {
				// Should not be able to occur because of bad node (no semicolon) check in NewGameTree
				panic("bug: node was not initialized before open bracket")
			}
			if gt.newIdentifier {
				for j := 0; j < len(gt.Nodes[i])-1; j++ {
					if ident == gt.Nodes[i][j].Identifier { // Check for duplicate identifier
						return fmt.Errorf("duplicate identifier: %s", ident)
					}
				}
				gt.Nodes[i] = append(gt.Nodes[i], Property{Identifier: ident})
				gt.newIdentifier = false
			} else if ident == "" {
				return fmt.Errorf("empty identifier")
			}

		} else if r == ']' { // End of a value
			i := len(gt.Nodes) - 1
			if i < 0 {
				// Should not be able to occur because of missing open bracket check in NewGameTree
				panic("bug: node was not initialized before close bracket")
			}
			j := len(gt.Nodes[i]) - 1
			if j < 0 {
				// Should not be able to occur because of missing open bracket check in NewGameTree
				panic("bug: property was not initialized before close bracket")
			}
			gt.Nodes[i][j].Values = append(gt.Nodes[i][j].Values, gt.currentValue.String())
			gt.currentValue.Reset()

		} else if unicode.IsUpper(r) { // Part of an identifier
			if gt.newIdentifier { // Part of current identifier
				gt.currentIdentifier.WriteRune(r)
			} else { // Start of a new identifier
				gt.currentIdentifier.Reset()
				gt.currentIdentifier.WriteRune(r)
				gt.newIdentifier = true
			}
		} else if !unicode.IsSpace(r) {
			return fmt.Errorf("unexpected identifier character: %q", r)
		}
	}
	return nil
}

// Equals checks GameTree equality recursively
func (gt *GameTree) Equals(gt2 *GameTree) bool {
	if len(gt.Nodes) != len(gt2.Nodes) {
		return false
	}
	for i := range gt.Nodes {
		if len(gt.Nodes[i]) != len(gt2.Nodes[i]) {
			return false
		}
		for j := range gt.Nodes[i] {
			if gt.Nodes[i][j].Identifier != gt2.Nodes[i][j].Identifier {
				return false
			}
			if len(gt.Nodes[i][j].Values) != len(gt2.Nodes[i][j].Values) {
				return false
			}
			for k := range gt.Nodes[i][j].Values {
				if gt.Nodes[i][j].Values[k] != gt2.Nodes[i][j].Values[k] {
					return false
				}
			}
		}
	}
	if len(gt.Children) != len(gt2.Children) {
		return false
	}
	for i := range gt.Children {
		if !gt.Children[i].Equals(gt2.Children[i]) {
			return false
		}
	}
	return true
}

func (gt GameTree) String() string {
	s := "\n"
	for _, node := range gt.Nodes {
		for _, prop := range node {
			s += prop.Identifier
			for _, v := range prop.Values {
				s += "{" + v + "}"
			}
			s += " "
		}
		s += "\n"
	}
	for i := range gt.Children {
		childString := gt.Children[i].String()
		s += strings.Replace(childString, "\n", "\n  ", -1)
	}
	return s
}
