package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/dodgebc/go-game-utils/sgfgrab"
	"golang.org/x/build/pargzip"
)

// loader loads files from a single .tar.gz archive, all errors are fatal
func loader(tgzFile string, cancel <-chan struct{}) (<-chan []byte, <-chan error) {
	out := make(chan []byte)
	cerr := make(chan error)

	go func() {
		defer close(out)
		defer close(cerr)

		// Open .tar.gz input file stream
		fin, err := os.Open(tgzFile)
		if err != nil {
			select {
			case cerr <- fmt.Errorf("failed to open tgz archive: %s", err):
			case <-cancel:
			}
			return
		}
		defer fin.Close()

		// Decompress input file stream
		gzipReader, err := gzip.NewReader(fin)
		if err != nil {
			select {
			case cerr <- fmt.Errorf("failed to decompress tgz archive: %s", err):
			case <-cancel:
			}
			return
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
				select {
				case cerr <- fmt.Errorf("tar archive read error: %s", err):
				case <-cancel:
				}
				return
			}

			if header.Typeflag == tar.TypeReg {
				// Is it an SGF file?
				hName := header.Name
				if (len(hName) >= 4) && (strings.ToLower(hName[len(hName)-4:]) == ".sgf") {
					sgfBytes, err := ioutil.ReadAll(tarReader) // Read whole file
					if err != nil {
						select {
						case cerr <- fmt.Errorf("sgf read error: %s", err):
						case <-cancel:
						}
						return
					}
					select {
					case out <- sgfBytes: // Send for processing
					case <-cancel:
						return
					}
				}
			}
		}
	}()
	return out, cerr
}

// parser converts incoming SGF data into json lines
func parser(in <-chan []byte, sourceName string, gameIDSource <-chan uint32, workers int, cancel <-chan struct{}) (<-chan []byte, <-chan error) {
	out := make(chan []byte)
	cerr := make(chan error)

	var wg sync.WaitGroup
	go func() {
		defer close(out)
		defer close(cerr)
		wg.Add(workers)
		defer wg.Wait()

		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()

				for sgfBytes := range in {
					var jsonGames []byte

					// Parse SGF
					games, err := sgfgrab.Grab(string(sgfBytes))
					if err != nil {
						select {
						case cerr <- err:
						case <-cancel:
							return
						}
					}

					// Convert to JSON
					for _, g := range games {
						g.Source = sourceName
						select {
						case g.GameID = <-gameIDSource:
						case <-cancel:
							return
						}
						j, err := json.Marshal(g)
						if err != nil {
							log.Fatalf("json marshal error: %s", err)
						}
						jsonGames = append(jsonGames, j...)
						jsonGames = append(jsonGames, '\n')
					}
					select {
					case out <- jsonGames:
					case <-cancel:
						return
					}
				}
			}()
		}
	}()
	return out, cerr
}

// saver gzips and saves json lines data to file, returns done channel, errors exit program
func saver(in <-chan []byte, outFile string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)

		// Open .jsonl.gz output file stream
		f, err := os.Create(outFile)
		if err != nil {
			log.Fatalf("failed to create output file: %s", err)
		}
		defer f.Close()

		// Compress output file stream
		gzipWriter := pargzip.NewWriter(f)
		defer gzipWriter.Close()

		for b := range in {
			_, err := gzipWriter.Write(b)
			if err != nil {
				log.Fatalf("write error: %s", err)
			}
		}
	}()
	return done
}
