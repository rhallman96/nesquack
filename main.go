package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/rhallman96/nesquack/gui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ROM filename was not provided")
		return
	}

	filename := os.Args[1]
	rom, err := load(filename)
	if err != nil {
		fmt.Println("failed to load rom from " + filename)
		os.Exit(1)
	}

	gui.Launch(rom)
}

func load(filename string) ([]uint8, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	r := make([]uint8, len(file))
	for i, v := range file {
		r[i] = uint8(v)
	}

	log.Printf("Loaded %s (%d bytes)", filename, len(r))

	return r, nil
}
