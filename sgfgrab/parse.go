package sgfgrab

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Pre-compile regular expressions for parsing
var reSquare, reRect, reResult, reRanks, reDate, reMove *regexp.Regexp

func init() {
	reSquare = regexp.MustCompile("^[0-9]{1,2}$")
	reRect = regexp.MustCompile("^[0-9]{1,2}:[0-9]{1,2}$")
	reResult = regexp.MustCompile("^([BW])\\+([0-9]+(?:\\.[0-9]+)?|R|Resign|T|Time|F|Forfeit|)$")
	reRanks = regexp.MustCompile("^[0-9]{1,2}[kdp]$")
	reDate = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")
	//reMove = regexp.MustCompile("^[a-zA-Z]{2}$")
}

// ErrParse means that a property was not able to be parsed
type ErrParse struct {
	identifier string
	value      string
}

func (e ErrParse) Error() string {
	return fmt.Sprintf("property parse error: %s %s", e.identifier, e.value)
}

// ParseSize parses board ROWS and COLUMNS
func ParseSize(v string) ([2]int, error) {
	switch {
	case reSquare.MatchString(v): // Square board
		size, _ := strconv.Atoi(v)
		return [2]int{size, size}, nil
	case reRect.MatchString(v): // Rectangular board
		dims := strings.Split(v, ":")
		if len(dims) > 2 {
			return [2]int{}, ErrParse{"SZ", v}
		}
		cols, _ := strconv.Atoi(dims[0])
		rows, _ := strconv.Atoi(dims[1])
		return [2]int{rows, cols}, nil
	}
	return [2]int{}, ErrParse{"SZ", v}
}

// ParseKomi parses komi
func ParseKomi(v string) (float64, error) {
	vFloat, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0.0, ErrParse{"KM", v}
	}
	weirdKomis := []float64{150, 250, 350, 450, 550, 650, 750} // Some games write, for example, 650 instead of 6.5
	for _, wk := range weirdKomis {
		if vFloat == wk {
			vFloat = wk / 100
		}
	}
	/*vString := fmt.Sprintf("%.1f", vFloat)
	reFloat := regexp.MustCompile("[0-9]+(\\.5)?")
	if !reFloat.MatchString(vString) {
		return 0.0, ErrParse{"KM", v}
	}*/
	return vFloat, nil
}

// ParseHandicap parses handicap
func ParseHandicap(v string) (int, error) {
	vInt, err := strconv.Atoi(v)
	if err != nil {
		return 0, ErrParse{"HA", v}
	}
	if vInt < 0 {
		return 0, ErrParse{"HA", v}
	}
	return vInt, nil
}

// ParseResult parses result into winner, score, and special ("Time", "Resign", or "Forfeit")
func ParseResult(v string) (string, float64, string, error) {
	for _, s := range []string{"", "0", "draw", "void", "?"} {
		if strings.ToLower(v) == s {
			return "", 0.0, "None", nil // No result
		}
	}
	findResult := reResult.FindStringSubmatch(v)
	if (findResult == nil) || (len(findResult) != 3) {
		return "", 0.0, "", ErrParse{"RE", v}
	}
	switch findResult[2] {
	case "R", "Resign":
		return findResult[1], 0.0, "Resign", nil
	case "T", "Time":
		return findResult[1], 0.0, "Time", nil
	case "F", "Forfeit":
		return findResult[1], 0.0, "Forfeit", nil
	case "":
		return findResult[1], 0.0, "None", nil
	}

	// Should be float now
	vFloat, err := strconv.ParseFloat(findResult[2], 64)
	if err != nil {
		return "", 0.0, "", ErrParse{"RE", v}
	}
	/*vString := fmt.Sprintf("%.1f", vFloat)
	reFloat := regexp.MustCompile("[0-9]+(\\.5)?")
	if !reFloat.MatchString(vString) {
		return "", 0.0, "", ErrParse{"RE", v}
	}*/
	return findResult[1], vFloat, "", nil
}

// ParseRank parses player rank ("B" or "W" else panic)
func ParseRank(player, v string) (string, error) {
	if (player != "B") && (player != "W") {
		panic("player was not black or white")
	}
	replacements := map[string]string{
		"级": "k", "段": "d", "a": "p",
		"-": "", "零": "0", "一": "1", "二": "2", "三": "3", "四": "4", "五": "5", "六": "6", "七": "7", "八": "8", "九": "9",
	}
	for s1, s2 := range replacements {
		v = strings.Replace(v, s1, s2, -1)
	}
	if strings.Contains(v, "P") { // Noticed that some games have P7d to mean 7p
		v = strings.Replace(v, "P", "", -1)
		v = strings.Replace(v, "d", "p", -1)
	}
	if reRanks.MatchString(v) {
		return v, nil
	}
	return "", ErrParse{player + "R", v}
}

// ParsePlayer parses player name ("B" or "W" else panic)
func ParsePlayer(player, v string) (string, error) {
	if (player != "B") && (player != "W") {
		panic("player was not black or white")
	}
	return v, nil
}

// ParseTime parses time limit in seconds
func ParseTime(v string) (int, error) {
	vInt, err := strconv.Atoi(v)
	if err != nil {
		return 0, ErrParse{"TM", v}
	}
	if vInt < 0 {
		return 0, ErrParse{"TM", v}
	}
	return vInt, nil
}

// ParseOvertime parses overtime (no filtering)
func ParseOvertime(v string) (string, error) {
	return v, nil
}

// ParseDate parses game date (either "" or formatted like "2020-01-01")
func ParseDate(v string) (string, error) {

	if !reDate.MatchString(v) {
		return "", ErrParse{"DT", v}
	}
	return v, nil
}

// ParseMove parses a game move in SGF format  ("B" or "W" else panic)
func ParseMove(player, v string) (string, error) {
	if (player != "B") && (player != "W") {
		panic("player was not black or white")
	}
	// Regex version is slow and this function is called very frequently
	/*
		if !reMove.MatchString(v) {
			return "", ErrParse{player, v}
		}
		return player + v, nil
	*/
	if len(v) == 0 {
		return player, nil
	}
	vr := []rune(v)
	if len(vr) == 2 {
		if unicode.IsLetter(vr[0]) && unicode.IsLetter(vr[1]) {
			return player + v, nil
		}
	}
	return "", ErrParse{player, v}
}
