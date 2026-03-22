package mister

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDIPPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Elevator Action.mra", "/media/fat/config/dips/Elevator Action.dip"},
		{"/media/fat/_Arcade/Elevator Action.mra", "/media/fat/config/dips/Elevator Action.dip"},
		{"game.mra", "/media/fat/config/dips/game.dip"},
	}
	for _, tt := range tests {
		got := DIPPath(tt.input)
		if got != tt.want {
			t.Errorf("DIPPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseMRADefaults(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"FF,FF,FF", []byte{0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"00,7F,00,FF,00,00,00,00", []byte{0x00, 0x7F, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00}},
		{"", make([]byte, 8)},
		{"AB,CD", []byte{0xAB, 0xCD, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
	}
	for _, tt := range tests {
		got := ParseMRADefaults(tt.input)
		if len(got) != DIPSize {
			t.Errorf("ParseMRADefaults(%q) len = %d, want %d", tt.input, len(got), DIPSize)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("ParseMRADefaults(%q)[%d] = %02x, want %02x", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestGetMRADefaults(t *testing.T) {
	mra, err := ParseMRAData([]byte(testMRA))
	if err != nil {
		t.Fatal(err)
	}

	defaults := GetMRADefaults(mra)
	// testMRA has default="FF,FF,FF"
	if defaults[0] != 0xFF || defaults[1] != 0xFF || defaults[2] != 0xFF {
		t.Errorf("defaults = %v, want [FF,FF,FF,00,...]", defaults)
	}
	// Rest should be zero
	for i := 3; i < DIPSize; i++ {
		if defaults[i] != 0 {
			t.Errorf("defaults[%d] = %02x, want 0x00", i, defaults[i])
		}
	}
}

func TestReadWriteDIP(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "test.dip")

	// Write DIP data
	data := []byte{0xFF, 0x7F, 0x00, 0xAB, 0x00, 0x00, 0x00, 0x00}
	if err := WriteDIP(dipPath, data); err != nil {
		t.Fatalf("WriteDIP failed: %v", err)
	}

	// Read it back
	got, err := ReadDIP(dipPath)
	if err != nil {
		t.Fatalf("ReadDIP failed: %v", err)
	}
	for i := range data {
		if got[i] != data[i] {
			t.Errorf("byte %d: got %02x, want %02x", i, got[i], data[i])
		}
	}
}

func TestWriteDIP_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "test.dip")

	// Create initial file
	if err := os.WriteFile(dipPath, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}

	// Write new data (should create backup)
	if err := WriteDIP(dipPath, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}); err != nil {
		t.Fatal(err)
	}

	// Verify backup exists
	entries, _ := os.ReadDir(tmpDir)
	bakFound := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.dip.bak-") {
			bakFound = true
			break
		}
	}
	if !bakFound {
		t.Error("WriteDIP should create a backup file")
	}
}

func TestWriteDIP_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "dips", "subdir", "test.dip")

	data := make([]byte, DIPSize)
	if err := WriteDIP(dipPath, data); err != nil {
		t.Fatalf("WriteDIP should create parent dirs: %v", err)
	}

	if _, err := os.Stat(dipPath); err != nil {
		t.Errorf("DIP file should exist: %v", err)
	}
}

func TestWriteDIP_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "new.dip")

	// Writing to a new file should not error (no backup needed)
	if err := WriteDIP(dipPath, make([]byte, DIPSize)); err != nil {
		t.Fatalf("WriteDIP to new file should succeed: %v", err)
	}
}

func TestLoadDIPData_NoFile(t *testing.T) {
	mra, err := ParseMRAData([]byte(testMRA))
	if err != nil {
		t.Fatal(err)
	}

	// No .dip file exists → should return MRA defaults
	data := LoadDIPData("/nonexistent/path.dip", mra)
	if data[0] != 0xFF || data[1] != 0xFF || data[2] != 0xFF {
		t.Errorf("LoadDIPData with no file should return MRA defaults, got %v", data)
	}
}

func TestLoadDIPData_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "test.dip")

	custom := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := os.WriteFile(dipPath, custom, 0644); err != nil {
		t.Fatal(err)
	}

	mra, _ := ParseMRAData([]byte(testMRA))
	data := LoadDIPData(dipPath, mra)
	for i := range custom {
		if data[i] != custom[i] {
			t.Errorf("LoadDIPData byte %d = %02x, want %02x", i, data[i], custom[i])
		}
	}
}

func TestDIPBitManipulationOnDefaults(t *testing.T) {
	// Simulate: MRA defaults are FF,FF,FF, then we toggle specific DIP bits
	mra, err := ParseMRAData([]byte(testMRA))
	if err != nil {
		t.Fatal(err)
	}

	dipData := GetMRADefaults(mra) // [0xFF, 0xFF, 0xFF, 0x00, ...]
	dips := ParseDIPSwitches(mra)

	// "Free Play" is bit 10, defaults to FF so bit 10 = 1
	fp := FindDIPSwitch(dips, "Free Play")
	if fp == nil {
		t.Fatal("expected Free Play DIP")
	}

	// Default value: bit 10 is set (in byte 1, bit 2) → 0xFF has all bits set
	val := GetBitRange(dipData, fp.Bit, fp.BitHigh)
	if val != 1 {
		t.Errorf("Free Play default = %d, want 1 (bit 10 set in 0xFF)", val)
	}

	// Set Free Play to 0 ("On" = index 0)
	SetBitRange(dipData, fp.Bit, fp.BitHigh, 0)
	val = GetBitRange(dipData, fp.Bit, fp.BitHigh)
	if val != 0 {
		t.Errorf("Free Play after set to 0 = %d, want 0", val)
	}

	// "Lives" is bits 1-2, default 0xFF means bits 1,2 = 1,1 = value 3
	lives := FindDIPSwitch(dips, "Lives")
	if lives == nil {
		t.Fatal("expected Lives DIP")
	}
	val = GetBitRange(dipData, lives.Bit, lives.BitHigh)
	if val != 3 {
		t.Errorf("Lives default = %d, want 3 (bits 1,2 set in 0xFF)", val)
	}

	// Set Lives to 1 ("4")
	SetBitRange(dipData, lives.Bit, lives.BitHigh, 1)
	val = GetBitRange(dipData, lives.Bit, lives.BitHigh)
	if val != 1 {
		t.Errorf("Lives after set to 1 = %d, want 1", val)
	}

	// Verify Free Play wasn't affected
	if v := GetBitRange(dipData, fp.Bit, fp.BitHigh); v != 0 {
		t.Errorf("Free Play changed after Lives write: %d, want 0", v)
	}
}

func TestDIPWriteReadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	dipPath := filepath.Join(tmpDir, "roundtrip.dip")

	mra, _ := ParseMRAData([]byte(testMRA))
	dipData := GetMRADefaults(mra)
	dips := ParseDIPSwitches(mra)

	// Modify several DIP switches
	fp := FindDIPSwitch(dips, "Free Play")
	SetBitRange(dipData, fp.Bit, fp.BitHigh, 0)

	coin := FindDIPSwitch(dips, "Coin A")
	SetBitRange(dipData, coin.Bit, coin.BitHigh, 2) // "2C/1P"

	// Write to disk
	if err := WriteDIP(dipPath, dipData); err != nil {
		t.Fatal(err)
	}

	// Read back
	readData := LoadDIPData(dipPath, mra)

	// Verify values survived round-trip
	if v := GetBitRange(readData, fp.Bit, fp.BitHigh); v != 0 {
		t.Errorf("Free Play after roundtrip = %d, want 0", v)
	}
	if v := GetBitRange(readData, coin.Bit, coin.BitHigh); v != 2 {
		t.Errorf("Coin A after roundtrip = %d, want 2", v)
	}
}
