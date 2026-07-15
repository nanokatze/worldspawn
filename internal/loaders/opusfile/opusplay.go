//go:build ignore

package main

import (
	"flag"
	"io"
	"log"
	"os"

	"worldspawn/internal/loaders/opusfile"
)

func main() {
	flag.Parse()

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r, err := opusfile.NewReader(f)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(r.Seek(48000*4*2, io.SeekCurrent))

	for {
		if _, err := io.CopyN(os.Stdout, r, 104729); err != nil {
			log.Fatal(err)
		}
	}

	// select {}
}
