package weiqi

import "fmt"

// Ruleset describes the rules as they pertain to ko and suicide
type rules struct {
	situationalSuperko bool
	positionalSuperko  bool
	suicideForbidden   bool
}

// Currently supports {"NZ", "AGA", "TT", ""}
func newRules(ruleset string) (rules, error) {
	var r rules
	switch ruleset {
	case "NZ":
		r.situationalSuperko = true
	case "AGA":
		r.situationalSuperko = true
		r.suicideForbidden = true
	case "TT":
		r.positionalSuperko = true
	case "":
	default:
		return r, fmt.Errorf("ruleset not supported: %v", ruleset)
	}
	return r, nil
}
