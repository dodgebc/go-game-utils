/*
Package sgf provides simple SGF parsing capabilities.*/
package sgf

// From the SGF spec:
/*
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

// Differences from SGF spec:

// Multiple identical identifiers in one node have all values collected
// In SGF spec, this should not be allowed (an error)

// Something like a(b)c will be parsed as ac(b)
// In SGF spec, this should not be allowed (an error)

// All whitespace is immediately converted to single space
// There are some exceptions/edge cases in the SGF spec

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Node contains key/value(s) pairs
type Node map[string][]string

// GameTree is a sequence of nodes potentially followed by other GameTrees (per SGF spec)
type GameTree struct {
	Nodes    []Node
	Children []*GameTree

	// state variables for parsing
	currentNode             Node
	currentIdentifierString string
	currentIdentifier       strings.Builder
	currentValue            strings.Builder
	escaped                 bool
	whitespace              bool
}

// NewGameTree parses text in SGF format into a GameTree
func NewGameTree(sgfText string) (GameTree, error) {

	// Parse setup
	brackOpen := false
	escaped := false
	var root GameTree
	var stack []*GameTree
	stack = append(stack, &root)

	// Iterative tree parsing (single pass)
	for _, r := range sgfText {

		// Mark property values
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
		}

		// Expand tree
		if isValue {
			err := stack[len(stack)-1].addRune(r, true)
			if err != nil {
				return GameTree{}, err
			}
		} else {
			if r == '(' {
				add := new(GameTree)
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, add)
				stack = append(stack, add)
			} else if r == ')' {
				if len(stack) > 1 {
					stack = stack[:len(stack)-1]
				} else {
					return GameTree{}, fmt.Errorf("missing open parenthesis")
				}
			} else {
				err := stack[len(stack)-1].addRune(r, false)
				if err != nil {
					return GameTree{}, err
				}
			}
		}
	}
	if brackOpen {
		return GameTree{}, fmt.Errorf("missing close bracket")
	}
	if len(stack) > 1 {
		return GameTree{}, fmt.Errorf("missing close parenthesis")
	}
	return root, nil
}

// Equals checks GameTree equality recursively
func (gt *GameTree) Equals(gt2 *GameTree) bool {
	if len(gt.Nodes) != len(gt2.Nodes) {
		return false
	}
	nodes := gt.Nodes
	nodes2 := gt2.Nodes
	for i := range nodes {
		for k := range nodes[i] {
			if len(nodes[i][k]) != len(nodes2[i][k]) {
				return false
			}
			for j := range nodes[i][k] {
				if nodes[i][k][j] != nodes2[i][k][j] {
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
	for i := range gt.Nodes {
		keys := []string{}
		for k := range gt.Nodes[i] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s += k
			for _, v := range gt.Nodes[i][k] {
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

func (gt *GameTree) addRune(r rune, isValue bool) error {
	if isValue { // Inside a value
		if unicode.IsSpace(r) && (r != ' ') { // Strip non-space whitespace
			if !gt.whitespace {
				gt.currentValue.WriteRune(' ')
			}
			gt.whitespace = true
		} else if (r == '\\') && !gt.escaped { // Don't record escape characters
			gt.escaped = true
			gt.whitespace = false
		} else {
			gt.currentValue.WriteRune(r) // Update current value
			gt.escaped = false
			gt.whitespace = false
		}
	} else { // Control rune
		gt.whitespace = false
		if gt.escaped {
			return fmt.Errorf("bug: escaped character outside of value")
		}
		if r == ';' { // Start of a new node
			if gt.currentValue.Len() != 0 {
				return fmt.Errorf("value did not end: %s", gt.currentValue.String())
			}
			gt.currentIdentifier.Reset()
			gt.currentNode = make(Node)
			gt.Nodes = append(gt.Nodes, gt.currentNode)
		} else if r == '[' { // Start of a value
			gt.currentIdentifierString = gt.currentIdentifier.String()
		} else if r == ']' { // End of a value
			id := gt.currentIdentifierString
			gt.currentNode[id] = append(gt.currentNode[id], gt.currentValue.String())
			gt.currentValue.Reset()
		} else if unicode.IsUpper(r) { // Part of an identifier
			if gt.currentIdentifierString == "" { // Update current identifier
				gt.currentIdentifier.WriteRune(r)
			} else { // Begin a new identifier
				gt.currentIdentifier.Reset()
				gt.currentIdentifier.WriteRune(r)
				gt.currentIdentifierString = ""
			}
		}
	}
	return nil
}
