package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"log"
	"math/rand"
	"os"

	"github.com/dodgebc/go-game-utils/sgfgrab"
)

func readDataset(inFile string, out chan<- sgfgrab.GameData, sample float64) {
	defer close(out)

	// Open input file
	f, err := os.Open(inFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decompress input file
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}
	defer gzipReader.Close()

	// Read and unmarshal jsonl
	scanner := bufio.NewScanner(gzipReader)
	src := rand.NewSource(1)
	random := rand.New(src)
	for scanner.Scan() {
		if (sample == 1.0) || (random.Float64() < sample) {
			g := sgfgrab.GameData{}
			err := json.Unmarshal(scanner.Bytes(), &g)
			if err != nil {
				log.Print(err)
			} else {
				out <- g
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func writeDataset(outFile string, in <-chan sgfgrab.GameData, done chan<- struct{}) {
	defer close(done)

	// Open output file
	f, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Compress output file
	gzipWriter := gzip.NewWriter(f)
	defer gzipWriter.Close()

	// Marshal and write jsonl
	for g := range in {
		j, err := json.Marshal(g)
		if err != nil {
			log.Print(err)
		} else {
			j = append(j, '\n')
			_, err := gzipWriter.Write(j)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
