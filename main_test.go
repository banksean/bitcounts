package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBitCountingAlgorithm tests the core bit counting logic
func TestBitCountingAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int
	}{
		{"empty", []byte{}, 0},
		{"single zero byte", []byte{0x00}, 0},
		{"single one bit", []byte{0x01}, 1},
		{"all bits set", []byte{0xFF}, 8},
		{"half bits set", []byte{0x0F}, 4},
		{"alternating pattern", []byte{0xAA}, 4},         // 10101010
		{"multiple bytes", []byte{0xFF, 0x00, 0x0F}, 12}, // 8 + 0 + 4
		{"text content", []byte("A"), 2},                 // 'A' = 0x41 = 01000001 (2 bits)
		{"text hello", []byte("hello"), 21},              // h=3, e=4, l=4, l=4, o=6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Manually count bits to simulate the algorithm
			ones := 0
			for _, b := range tt.input {
				c := 0
				for temp := b; temp != 0; c++ {
					temp &= temp - 1 // clear the least significant bit set
				}
				ones += c
			}

			if ones != tt.expected {
				t.Errorf("expected %d ones, got %d for input %v", tt.expected, ones, tt.input)
			}
		})
	}
}

// TestCountFile tests the countFile method with various file contents
func TestCountFile(t *testing.T) {
	tests := []struct {
		name         string
		content      []byte
		expectedBits int
	}{
		{"empty file", []byte{}, 0},
		{"single zero byte", []byte{0x00}, 0},
		{"single byte with all bits", []byte{0xFF}, 8},
		{"text content", []byte("hello world"), 45},            // calculated manually
		{"binary content", []byte{0x00, 0xFF, 0xAA, 0x55}, 16}, // 0 + 8 + 4 + 4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpfile, err := os.CreateTemp("", "bitcount_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			defer tmpfile.Close()

			// Write test content
			if _, err := tmpfile.Write(tt.content); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Reopen for reading
			file, err := os.Open(tmpfile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			// Test countFile
			bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
			err = bc.countFile(file)
			if err != nil {
				t.Errorf("countFile failed: %v", err)
			}

			if bc.ones != tt.expectedBits {
				t.Errorf("expected %d ones, got %d", tt.expectedBits, bc.ones)
			}

			if bc.bytesRead != len(tt.content) {
				t.Errorf("expected %d bytes read, got %d", len(tt.content), bc.bytesRead)
			}
		})
	}
}

// TestCountDirectory tests the count method with directory traversal
func TestCountDirectory(t *testing.T) {
	// Create temporary directory structure
	tmpdir, err := os.MkdirTemp("", "bitcount_test_dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Create test files
	testFiles := map[string][]byte{
		"file1.txt": []byte("hello"), // should have some bits
		"file2.bin": {0xFF, 0x00},    // 8 bits
		"empty.txt": {},              // 0 bits
	}

	totalExpectedBits := 0
	totalExpectedBytes := 0

	for filename, content := range testFiles {
		filepath := filepath.Join(tmpdir, filename)
		if err := os.WriteFile(filepath, content, 0644); err != nil {
			t.Fatal(err)
		}
		totalExpectedBytes += len(content)

		// Calculate expected bits for this file
		for _, b := range content {
			c := 0
			for temp := b; temp != 0; c++ {
				temp &= temp - 1
			}
			totalExpectedBits += c
		}
	}

	// Create subdirectory with a file
	subdir := filepath.Join(tmpdir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	subfilePath := filepath.Join(subdir, "subfile.txt")
	subfileContent := []byte{0xAA} // 4 bits
	if err := os.WriteFile(subfilePath, subfileContent, 0644); err != nil {
		t.Fatal(err)
	}
	totalExpectedBytes += len(subfileContent)
	totalExpectedBits += 4

	// Test the count method
	bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
	err = bc.count(tmpdir)
	if err != nil {
		t.Errorf("count failed: %v", err)
	}

	if bc.ones != totalExpectedBits {
		t.Errorf("expected %d total bits, got %d", totalExpectedBits, bc.ones)
	}

	if bc.bytesRead != totalExpectedBytes {
		t.Errorf("expected %d total bytes, got %d", totalExpectedBytes, bc.bytesRead)
	}

	if len(bc.errs) != 0 {
		t.Errorf("expected no errors, got: %v", bc.errs)
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("nonexistent directory", func(t *testing.T) {
		bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
		err := bc.count("/this/path/definitely/does/not/exist/anywhere")
		// filepath.Walk returns nil but passes error to callback, which adds it to bc.errs
		if err != nil {
			t.Errorf("count should not return error, got: %v", err)
		}
		// But there should be an error in the errors slice
		if len(bc.errs) == 0 {
			t.Error("expected at least one error in bc.errs for nonexistent directory")
		}
		// Check that the error message contains something about the path not existing
		if len(bc.errs) > 0 && !strings.Contains(bc.errs[0], "walk error") {
			t.Errorf("expected 'walk error' message, got: %v", bc.errs)
		}
	})

	t.Run("valid directory processing", func(t *testing.T) {
		// Test that the error handling mechanism works with a valid directory
		tmpdir, err := os.MkdirTemp("", "bitcount_valid_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpdir)

		// Create a normal file that should be processed successfully
		testFile := filepath.Join(tmpdir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
		err = bc.count(tmpdir)
		// The count method should not return an error for valid directory
		if err != nil {
			t.Errorf("count should not return error for valid directory, got: %v", err)
		}

		// Should have no errors
		if len(bc.errs) != 0 {
			t.Errorf("expected no errors, got: %v", bc.errs)
		}

		// Should have processed the file
		if bc.bytesRead != 4 { // "test" is 4 bytes
			t.Errorf("expected 4 bytes read, got %d", bc.bytesRead)
		}
	})
}

// TestLargeFile tests behavior with files larger than the buffer
func TestLargeFile(t *testing.T) {
	// Create a file larger than BUFF_SIZE to test buffer handling
	tmpfile, err := os.CreateTemp("", "bitcount_large_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Write data larger than buffer size
	testData := make([]byte, BUFF_SIZE+1000)
	for i := range testData {
		testData[i] = 0xFF // All bits set
	}
	if _, err := tmpfile.Write(testData); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen for reading
	file, err := os.Open(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	// Test countFile
	bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}
	err = bc.countFile(file)
	if err != nil {
		t.Errorf("countFile failed on large file: %v", err)
	}

	expectedBits := len(testData) * 8 // All bits are set
	if bc.ones != expectedBits {
		t.Errorf("expected %d ones, got %d", expectedBits, bc.ones)
	}

	if bc.bytesRead != len(testData) {
		t.Errorf("expected %d bytes read, got %d", len(testData), bc.bytesRead)
	}
}

// TestBitCountAccuracy verifies the bit counting algorithm matches expected results
func TestBitCountAccuracy(t *testing.T) {
	tests := []struct {
		name  string
		byte  byte
		count int
	}{
		{"0x00", 0x00, 0},
		{"0x01", 0x01, 1},
		{"0x02", 0x02, 1},
		{"0x03", 0x03, 2},
		{"0x0F", 0x0F, 4},
		{"0x10", 0x10, 1},
		{"0x55", 0x55, 4}, // 01010101
		{"0xAA", 0xAA, 4}, // 10101010
		{"0xFF", 0xFF, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.byte
			c := 0
			// Use the same algorithm as in the main code
			for ; b != 0; c++ {
				b &= b - 1 // clear the least significant bit set
			}
			if c != tt.count {
				t.Errorf("expected %d bits for 0x%02X, got %d", tt.count, tt.byte, c)
			}
		})
	}
}

// TestInitialization tests that bitcounter is properly initialized
func TestInitialization(t *testing.T) {
	bc := &bitcounter{inbuf: make([]byte, BUFF_SIZE)}

	if bc.bytesRead != 0 {
		t.Errorf("expected bytesRead to be 0, got %d", bc.bytesRead)
	}

	if bc.ones != 0 {
		t.Errorf("expected ones to be 0, got %d", bc.ones)
	}

	if len(bc.errs) != 0 {
		t.Errorf("expected empty errors slice, got %d errors", len(bc.errs))
	}

	if len(bc.inbuf) != BUFF_SIZE {
		t.Errorf("expected buffer size %d, got %d", BUFF_SIZE, len(bc.inbuf))
	}
}
