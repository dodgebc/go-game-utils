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
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// GameTree is a sequence of nodes potentially followed by other GameTrees (per SGF spec)
type GameTree struct {
	Sequence Sequence
	Children []*GameTree
}

// NewGameTree parses text in SGF format into a GameTree
func NewGameTree(sgfText string) (GameTree, error) {
	re := regexp.MustCompile(`[^\S ]+`)
	// Using regexp might be a little slow... idea is to have just a SINGLE pass
	// Can easily do this in loop below if necessary
	sgfRunes := []rune(re.ReplaceAllString(sgfText, " ")) // for easy character indexing

	// Parse setup
	brackOpen := false
	escaped := false
	var root GameTree
	var stack []*GameTree
	stack = append(stack, &root)

	// Iterative tree parsing (single pass)
	for _, r := range sgfRunes {

		// Mark property values
		isValue := false
		switch {
		case brackOpen:
			isValue = true
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == ']' {
				brackOpen = false
				isValue = false
			}
		case r == '[':
			brackOpen = true
		}

		// Expand tree
		switch {
		case !isValue:
			if r == '(' {
				add := new(GameTree)
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, add)
				stack = append(stack, add)
			} else if r == ')' {
				if len(stack) > 1 {
					err := stack[len(stack)-1].Sequence.finish()
					if err != nil {
						return GameTree{}, err
					}
					stack = stack[:len(stack)-1]
				} else {
					return GameTree{}, fmt.Errorf("missing open parenthesis")
				}
			} else {
				err := stack[len(stack)-1].Sequence.addRune(r, false)
				if err != nil {
					return GameTree{}, err
				}
			}
		case isValue:
			err := stack[len(stack)-1].Sequence.addRune(r, true)
			if err != nil {
				return GameTree{}, err
			}
		}
	}
	if brackOpen {
		return GameTree{}, fmt.Errorf("missing close bracket")
	}
	if len(stack) > 1 {
		return GameTree{}, fmt.Errorf("missing close parenthesis")
	}
	stack[0].Sequence.finish()
	return root, nil
}

// Equals checks GameTree equality recursively
func (gt *GameTree) Equals(gt2 *GameTree) bool {
	if len(gt.Sequence.Nodes) != len(gt2.Sequence.Nodes) {
		return false
	}
	nodes := gt.Sequence.Nodes
	nodes2 := gt2.Sequence.Nodes
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
	for i := range gt.Sequence.Nodes {
		keys := []string{}
		for k := range gt.Sequence.Nodes[i] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s += k
			for _, v := range gt.Sequence.Nodes[i][k] {
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

// Node contains key/value(s) pairs
type Node map[string][]string

// Sequence contains Nodes (per SGF spec)
type Sequence struct {
	Nodes         []Node // Each identifier may only appear once in SGF, but can have multiple values
	runeBuffer    []rune
	isValueBuffer []bool
	started       bool
}

// CHANGE: Should put the addNode code in here, this will streamline things A LOT.
// No need for finish() then either.
func (s *Sequence) addRune(r rune, isValue bool) error {
	defer func() { s.started = true }()
	if (r == ';') && !isValue && s.started { // Beginning of next node
		err := s.addNode(s.runeBuffer, s.isValueBuffer)
		if err != nil {
			return err
		}
		s.runeBuffer = s.runeBuffer[:0]
		s.isValueBuffer = s.isValueBuffer[:0]
	} else if ((r != ';') && !isValue) || isValue { // Interior of node
		s.runeBuffer = append(s.runeBuffer, r)
		s.isValueBuffer = append(s.isValueBuffer, isValue)
	}
	return nil
}

func (s *Sequence) finish() error {
	if s.started {
		err := s.addNode(s.runeBuffer, s.isValueBuffer)
		if err != nil {
			return err
		}
	}
	s.runeBuffer = s.runeBuffer[:0]
	s.isValueBuffer = s.isValueBuffer[:0]
	return nil
}

func (s *Sequence) addNode(content []rune, isValue []bool) error {
	node := make(map[string][]string)
	identifier := ""
	escaped := false // We don't want escape artifacts in our final output

	for i, r := range content { // This should be the stuff between semicolons
		switch {
		case !isValue[i] && !unicode.IsSpace(r) && (r != '[') && (r != ']'): // Part of identifier we care about
			if len(node[identifier]) == 0 {
				// No values seen yet (we are still reading through an identifier)
				identifier += string(r)
			} else {
				// Already saw values, we should keep them with the previous identifier and start a new one
				identifier = string(r)
			}
		case !isValue[i] && (r == '['): // Signifies start of a new value
			node[identifier] = append(node[identifier], "")
			escaped = false
		case isValue[i]: // Add value runes (except for escaping backslash)
			if !escaped && (r == '\\') {
				escaped = true
			} else {
				node[identifier][len(node[identifier])-1] += string(r) // Slow
				escaped = false
			}
		}
	}
	s.Nodes = append(s.Nodes, node)
	return nil
}
