package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type bitcounter struct {
	bytesRead, ones int
}

func (bc *bitcounter) countFile(infile *os.File) error {
	inbuf := make([]byte, 1000)
	reader := bufio.NewReader(infile)
	var err error
	bread := 0
	for ; err != io.EOF; bread, err = reader.Read(inbuf) {
		if err != nil {
			return err
		}
		for _, b := range inbuf[:bread] {
			c := 0
			// I am not this clever. Borrowed from https://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetKernighan
			for ; b != 0; c++ {
				b &= b - 1 // clear the least significant bit set
			}
			bc.ones += c
		}
		bc.bytesRead += bread
	}
	return nil
}

func (bc *bitcounter) count(root string) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Printf("%s\n", path)
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			bc.countFile(in)
		}
		return nil
	})
	return err
}

func main() {
	bc := &bitcounter{}
	if err := bc.count("."); err != nil {
		panic(err)
	}

	total := bc.bytesRead * 8

	fmt.Printf("total bits in input: %d. %d (%.2f%%) ones, %d (%.2f%%) zeroes.\n", total, bc.ones, (float64(100*bc.ones) / float64(total)), total-bc.ones, 100*float64(total-bc.ones)/float64(total))
}
