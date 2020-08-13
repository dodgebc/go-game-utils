package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"golang.org/x/build/pargzip"
)

func unmarshalGame(in <-chan []byte, workers int) <-chan sgfgrab.GameData {
	out := make(chan sgfgrab.GameData)
	var wg sync.WaitGroup
	go func() {
		defer close(out)
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for b := range in {
					g := sgfgrab.GameData{}
					err := json.Unmarshal(b, &g)
					if err != nil {
						log.Fatalf("failed to unmarshal json: %s", err)
					}
					out <- g
				}
			}()
		}
	}()
	return out
}

func marshalGame(in <-chan sgfgrab.GameData, workers int) <-chan []byte {
	out := make(chan []byte)
	var wg sync.WaitGroup
	go func() {
		defer close(out)
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for g := range in {
					b, err := json.Marshal(g)
					if err != nil {
						log.Fatalf("failed to marshal json: %s", err)
					}
					out <- b
				}
			}()
		}
	}()
	return out
}

func readGzipLines(inFile string, sample float64) <-chan []byte {
	out := make(chan []byte)
	go func() {
		defer close(out)

		// Open input file
		f, err := os.Open(inFile)
		if err != nil {
			log.Fatalf("failed to open input file: %s", err)
		}
		defer f.Close()

		// Decompress input file
		gzipReader, err := gzip.NewReader(f)
		if err != nil {
			log.Fatalf("failed to decompress input file: %s", err)
		}
		defer gzipReader.Close()

		// Read lines
		src := rand.NewSource(1)
		random := rand.New(src)
		scanner := bufio.NewScanner(gzipReader)
		for scanner.Scan() {
			if (sample == 1.0) || (random.Float64() < sample) {
				b := make([]byte, len(scanner.Bytes()))
				copy(b, scanner.Bytes())
				out <- b
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("input file read error: %s", err)
		}
	}()
	return out
}

func writeGzipLines(in <-chan []byte, outFile string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Open output file
		f, err := os.Create(outFile)
		if err != nil {
			log.Fatalf("failed to create output file: %s", err)
		}
		defer f.Close()

		// Compress output file
		gzipWriter := pargzip.NewWriter(f)
		defer gzipWriter.Close()

		// Write lines
		for b := range in {
			_, err := gzipWriter.Write(b)
			if err != nil {
				log.Fatalf("output file write error: %s", err)
			}
			_, err = gzipWriter.Write([]byte{'\n'})
			if err != nil {
				log.Fatalf("output file write error: %s", err)
			}
		}
	}()
	return done
}
