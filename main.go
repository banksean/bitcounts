package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

const BUFF_SIZE = 1000000

type bitcounter struct {
	bytesRead, ones int
	errs            []string
	inbuf           []byte
}

func (bc *bitcounter) countFile(infile *os.File) error {
	reader := bufio.NewReader(infile)
	var err error
	bread := 0
	for ; err != io.EOF; bread, err = reader.Read(bc.inbuf) {
		if err != nil {
			return err
		}
		for _, b := range bc.inbuf[:bread] {
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
	fsys := os.DirFS(root)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			bc.errs = append(bc.errs, fmt.Sprintf("walk error for %s: %v", path, err))
			return nil
		}
		if d == nil {
			bc.errs = append(bc.errs, fmt.Sprintf("nil fs.DirEntry: %s", path))
			return nil
		}
		if d.Type().IsRegular() {
			// Convert relative path back to full path for display and opening
			fullPath := root
			if path != "." {
				fullPath = root + "/" + path
			}
			fmt.Printf("%s\n", fullPath)
			in, err := os.Open(fullPath)
			if err != nil {
				bc.errs = append(bc.errs, err.Error())
				return nil
			}
			defer in.Close()
			if countErr := bc.countFile(in); countErr != nil {
				bc.errs = append(bc.errs, fmt.Sprintf("error counting file %s: %v", fullPath, countErr))
			}
		}
		return nil
	})
	return err
}

func main() {
	bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
	if err := bc.count("."); err != nil {
		panic(err)
	}

	total := bc.bytesRead * 8

	fmt.Printf("%d errors\n%v\n", len(bc.errs), strings.Join(bc.errs, "\n"))

	fmt.Printf("total bits in input: %d. %d (%.2f%%) ones, %d (%.2f%%) zeroes.\n", total, bc.ones, (float64(100*bc.ones) / float64(total)), total-bc.ones, 100*float64(total-bc.ones)/float64(total))
}
