package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"golang.org/x/build/pargzip"
)

func readTgzSgf(tgzFile string) <-chan []byte {
	out := make(chan []byte)

	go func() {
		defer close(out)

		// Open .tar.gz input file stream
		fin, err := os.Open(tgzFile)
		if err != nil {
			log.Fatalf("failed to open tgz archive: %s", err)
		}
		defer fin.Close()

		// Decompress input file stream
		gzipReader, err := gzip.NewReader(fin)
		if err != nil {
			log.Fatal("failed to decompress tgz archive: %s", err)
		}
		defer gzipReader.Close()

		// Read from archive and send data for processing
		tarReader := tar.NewReader(gzipReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				log.Fatal("tar archive read error: %s", err)
			}

			if header.Typeflag == tar.TypeReg {
				// Is it an SGF file?
				hName := header.Name
				if (len(hName) >= 4) && (strings.ToLower(hName[len(hName)-4:]) == ".sgf") {
					sgfBytes, err := ioutil.ReadAll(tarReader) // Read whole file
					if err != nil {
						log.Fatal("tar archive file read error: %s", err)
					}
					out <- sgfBytes // Send for processing
				}
			}
		}
	}()
	return out
}

func parseGame(in <-chan []byte, tgzName string, workers int) <-chan packet {
	out := make(chan packet)

	go func() {
		defer close(out)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				// Parse SGF
				for sgfBytes := range in {
					games, err := sgfgrab.Grab(string(sgfBytes))
					for _, g := range games {
						out <- packet{game: g, err: err, tgzName: tgzName}
					}
				}
			}()
		}
	}()
	return out
}

func marshalGame(in <-chan packet, workers int) <-chan []byte {
	out := make(chan []byte)

	go func() {
		defer close(out)
		var wg sync.WaitGroup
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for p := range in {
					b, err := json.Marshal(p.game)
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
