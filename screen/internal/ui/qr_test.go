package ui

import "testing"

// TestRSEncode checks the Reed-Solomon coder against the worked example of
// the standard (a 1-M symbol: 16 data codewords, 10 error-correction ones).
func TestRSEncode(t *testing.T) {
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	got := rsEncode(data, 10)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ec codewords %v, want %v", got, want)
		}
	}
}

// TestQRMatrixStructure checks the fixed parts of a symbol: size, finder
// patterns, the timing rows, the dark module and a valid format word.
func TestQRMatrixStructure(t *testing.T) {
	for _, text := range []string{"ABC123", "https://callme.rode.com/s?c=ABCDEF", "https://account.rode.com", string(make([]byte, 200))} {
		m := QRMatrix(text)
		if m == nil {
			t.Fatalf("%q: no matrix", text)
		}
		n := len(m)
		if (n-17)%4 != 0 || n < 21 || n > 57 {
			t.Fatalf("%q: size %d", text, n)
		}
		for _, c := range [][2]int{{3, 3}, {n - 4, 3}, {3, n - 4}} {
			for dy := -3; dy <= 3; dy++ {
				for dx := -3; dx <= 3; dx++ {
					d := max(abs(dx), abs(dy))
					want := d != 2
					if m[c[1]+dy][c[0]+dx] != want {
						t.Fatalf("%q: finder at %v broken at %d,%d", text, c, dx, dy)
					}
				}
			}
		}
		for i := 8; i < n-8; i++ {
			if m[6][i] != (i%2 == 0) || m[i][6] != (i%2 == 0) {
				t.Fatalf("%q: timing pattern broken at %d", text, i)
			}
		}
		if !m[n-8][8] {
			t.Fatalf("%q: dark module missing", text)
		}
		// The format word must be one of the eight level-M words, read from
		// both copies.
		read := 0
		for i := 0; i <= 5; i++ {
			if m[8][i] {
				read |= 1 << uint(i)
			}
		}
		if m[8][7] {
			read |= 1 << 6
		}
		if m[8][8] {
			read |= 1 << 7
		}
		if m[7][8] {
			read |= 1 << 8
		}
		for i := 9; i < 15; i++ {
			if m[14-i][8] {
				read |= 1 << uint(i)
			}
		}
		second := 0
		for i := 0; i < 7; i++ {
			if m[n-1-i][8] {
				second |= 1 << uint(i)
			}
		}
		for i := 7; i < 15; i++ {
			if m[8][n-15+i] {
				second |= 1 << uint(i)
			}
		}
		if read != second {
			t.Fatalf("%q: the two format copies differ: %015b vs %015b", text, read, second)
		}
		ok := false
		for mask := 0; mask < 8; mask++ {
			if qrFormatBits(mask) == read {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("%q: format word %015b is not a level-M word", text, read)
		}
	}
	if QRMatrix(string(make([]byte, 300))) != nil {
		t.Fatal("300 bytes do not fit version 10 at level M")
	}
}

// TestQRFormatBits checks the format word against the standard's example
// (level M, mask 5 → 100000011001110).
func TestQRFormatBits(t *testing.T) {
	if got := qrFormatBits(5); got != 0x40ce {
		t.Fatalf("format bits for M/5: %015b, want 100000011001110", got)
	}
}
