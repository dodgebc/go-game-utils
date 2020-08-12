package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"sort"
)

func loadBlacklist(blacklistFile string) []*regexp.Regexp {
	var blacklist []*regexp.Regexp
	f, err := os.Open(blacklistFile)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) > 0 {
			re, err := regexp.Compile("(?i)" + scanner.Text())
			if err != nil {
				log.Fatalf("failed to compile blacklist regexp: %s", err)
			}
			blacklist = append(blacklist, re)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return blacklist
}

func computeTopCut(topCut float64, playerCounts map[string]int) []string {
	var counts []int
	var names []string
	for _, c := range playerCounts {
		counts = append(counts, c)
	}
	sort.Ints(counts)
	cutoff := counts[int(float64(len(counts)-1)*(1-topCut))]
	for n, c := range playerCounts {
		if c > cutoff {
			names = append(names, n)
		}
	}
	return names
}
