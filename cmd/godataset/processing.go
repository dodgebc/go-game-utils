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

	"github.com/dodgebc/go-utils/sgfgrab"
)

// loader loads files from a single .tar.gz archive
func loader(cancel <-chan struct{}, cerr chan<- error, tgzFile string, out chan<- []byte, checker *CheckManager, progressUpdate *ProgressUpdate) {

	// Open .tar.gz input file stream
	fin, err := os.Open(tgzFile)
	if err != nil {
		select {
		case cerr <- err:
		case <-cancel:
		}
		return
	}
	defer fin.Close()

	// Decompress input file stream
	gzipReader, err := gzip.NewReader(fin)
	if err != nil {
		select {
		case cerr <- err:
		case <-cancel:
		}
		return
	}
	defer gzipReader.Close()

	// Read from archive and send data for processing
	tarReader := tar.NewReader(gzipReader)

	for {
		// Read SGF data
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			select {
			case cerr <- err:
			case <-cancel:
			}
			return
		}
		hName := header.Name
		if (header.Typeflag == tar.TypeReg) && (len(hName) >= 4) && (strings.ToLower(hName[len(hName)-4:]) == ".sgf") { // Is an SGF file
			sgfBytes, err := ioutil.ReadAll(tarReader) // Read whole file
			if err != nil {
				select {
				case cerr <- err:
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

		// Update progress
		progressUpdate.Update(1)
		progressUpdate.SetOther("failed", checker.NumFailed)
		if checker.MinLength > 0 {
			progressUpdate.SetOther("short", checker.NumShort)
		}
		if checker.CheckLegal {
			progressUpdate.SetOther("illegal", checker.NumIllegal)
		}
		if checker.Deduplicate {
			progressUpdate.SetOther("duplicate", checker.NumDuplicate)
		}
	}
}

// processor parses incoming SGF data to json lines
func processor(cancel <-chan struct{}, cerr chan<- error, in <-chan []byte, out chan<- []byte, checker *CheckManager, sourceName string) {
	for sgfBytes := range in {
		var jsonGames []byte

		// Parse SGF
		games, err := sgfgrab.Grab(string(sgfBytes))
		if err != nil {
			checker.AddFailed(1)
			if checker.Verbose {
				log.Print(err)
			}
		}

		// Check and convert to JSON
		for _, g := range games {
			if err := checker.Check(g); err != nil {
				if checker.Verbose {
					log.Print(err)
				}
			} else {
				g.Source = sourceName
				j, err := json.Marshal(g)
				if err != nil {
					select {
					case cerr <- err:
					case <-cancel:
					}
					return
				}
				jsonGames = append(jsonGames, j...)
				jsonGames = append(jsonGames, '\n')
			}
		}
		select {
		case out <- jsonGames:
		case <-cancel:
			return
		}
	}
}

// saver gzips and saves json lines data to a Writer
func saver(cancel <-chan struct{}, cerr chan<- error, in <-chan []byte, out io.Writer) {
	for b := range in {
		_, err := out.Write(b)
		if err != nil {
			select {
			case cerr <- err:
			case <-cancel:
			}
			return
		}
	}
}
