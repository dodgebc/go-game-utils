// Package sgfgrab provides lightweight scraping for SGF files.
package sgfgrab

import (
	"fmt"
	"strings"
	"unicode"
)

// Grab scrapes an SGF for GameData fields
func Grab(sgfText string) ([]GameData, error) {

	var brackOpen bool
	var parensOpen int
	var escaped bool
	var mainBranch bool

	var identifier strings.Builder
	var value strings.Builder
	var game GameData
	var allGames []GameData

	for _, r := range sgfText {

		// Manage brackets and escapes
		isValue := false
		isIdent := true
		justDone := false
		if brackOpen {
			isValue = true
			isIdent = false
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == ']' {
				brackOpen = false
				isValue = false
				justDone = true
			}
		} else if r == '[' {
			brackOpen = true
			isIdent = false
		} else if r == ']' {
			return []GameData{}, fmt.Errorf("missing open bracket")
		}

		// Manage parentheses
		if !isValue {
			if r == '(' {
				if parensOpen == 0 {
					mainBranch = true
				}
				parensOpen++
			} else if r == ')' {
				parensOpen--
				mainBranch = false
				if parensOpen == 0 {
					err := game.Finalize()
					if err != nil {
						return []GameData{}, err
					}
					allGames = append(allGames, game)
					game = GameData{}
				}
			}
		}

		// Don't worry about variations
		if mainBranch {

			if isValue { // Update value
				if value.Len() < 30 { // Probably an irrelevant comment, truncate
					value.WriteRune(r)
				}
			} else if isIdent && unicode.IsUpper(r) { // Update identifier
				identifier.WriteRune(r)
			} else if justDone { // Push property to GameData
				err := game.AddProperty(identifier.String(), value.String())
				if err != nil {
					return []GameData{}, err
				}
				value.Reset()
				identifier.Reset()
			}

		}

	}

	return allGames, nil
}
