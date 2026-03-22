package mister

import (
	"os"
	"path/filepath"
	"testing"
)

const testMRA = `<?xml version="1.0" encoding="utf-8"?>
<misterromdescription>
	<name>Elevator Action (bootleg)</name>
	<setname>elevatorb</setname>
	<rbf>TaitoSJ</rbf>
	<switches default="FF,FF,FF">
		<dip bits="0" name="Bonus Life" ids="30000,10000"/>
		<dip bits="1,2" name="Lives" ids="3,4,5,6"/>
		<dip bits="3" name="Unknown" ids="Off,On"/>
		<dip bits="6" name="Difficulty" ids="Easy,Hard"/>
		<dip bits="7" name="Cabinet" ids="Upright,Cocktail"/>
		<dip bits="10" name="Free Play" ids="On,Off"/>
		<dip bits="11" name="Demo Sounds" ids="On,Off"/>
		<dip bits="14,15" name="Coin A" ids="1C/1P,1C/2P,2C/1P,2C/3P"/>
	</switches>
</misterromdescription>`

func TestParseMRAData(t *testing.T) {
	mra, err := ParseMRAData([]byte(testMRA))
	if err != nil {
		t.Fatalf("ParseMRAData failed: %v", err)
	}

	if mra.SetName != "elevatorb" {
		t.Errorf("SetName = %q, want %q", mra.SetName, "elevatorb")
	}
}

func TestParseMRAFile(t *testing.T) {
	tmpDir := t.TempDir()
	mraPath := filepath.Join(tmpDir, "Elevator Action.mra")
	if err := os.WriteFile(mraPath, []byte(testMRA), 0644); err != nil {
		t.Fatal(err)
	}

	mra, err := ParseMRA(mraPath)
	if err != nil {
		t.Fatalf("ParseMRA failed: %v", err)
	}
	if mra.SetName != "elevatorb" {
		t.Errorf("SetName = %q, want %q", mra.SetName, "elevatorb")
	}
}

func TestParseDIPSwitches(t *testing.T) {
	mra, err := ParseMRAData([]byte(testMRA))
	if err != nil {
		t.Fatal(err)
	}

	dips := ParseDIPSwitches(mra)
	if len(dips) == 0 {
		t.Fatal("expected DIP switches, got none")
	}

	// Check "Free Play" DIP
	fp := FindDIPSwitch(dips, "Free Play")
	if fp == nil {
		t.Fatal("expected to find Free Play DIP switch")
	}
	if fp.Bit != 10 || fp.BitHigh != 10 {
		t.Errorf("Free Play bits = %d-%d, want 10-10", fp.Bit, fp.BitHigh)
	}
	if len(fp.Values) != 2 || fp.Values[0] != "On" || fp.Values[1] != "Off" {
		t.Errorf("Free Play values = %v, want [On, Off]", fp.Values)
	}

	// Check "Lives" DIP (multi-bit: 1,2)
	lives := FindDIPSwitch(dips, "Lives")
	if lives == nil {
		t.Fatal("expected to find Lives DIP switch")
	}
	if lives.Bit != 1 || lives.BitHigh != 2 {
		t.Errorf("Lives bits = %d-%d, want 1-2", lives.Bit, lives.BitHigh)
	}
	if len(lives.Values) != 4 {
		t.Errorf("Lives values count = %d, want 4", len(lives.Values))
	}

	// Check "Coin A" DIP (multi-bit: 14,15)
	coin := FindDIPSwitch(dips, "Coin A")
	if coin == nil {
		t.Fatal("expected to find Coin A DIP switch")
	}
	if coin.Bit != 14 || coin.BitHigh != 15 {
		t.Errorf("Coin A bits = %d-%d, want 14-15", coin.Bit, coin.BitHigh)
	}
}

func TestFindDIPValue(t *testing.T) {
	dip := &DIPSwitch{
		Name:   "Free Play",
		Bit:    10,
		Values: []string{"On", "Off"},
	}

	if idx := FindDIPValue(dip, "On"); idx != 0 {
		t.Errorf("On should be index 0, got %d", idx)
	}
	if idx := FindDIPValue(dip, "off"); idx != 1 { // case-insensitive
		t.Errorf("Off should be index 1, got %d", idx)
	}
	if idx := FindDIPValue(dip, "ZZ"); idx != -1 {
		t.Errorf("ZZ should be -1, got %d", idx)
	}
}

func TestParseDIPBits(t *testing.T) {
	tests := []struct {
		input    string
		wantLow  int
		wantHigh int
	}{
		{"10", 10, 10},
		{"1,2", 1, 2},
		{"14,15", 14, 15},
		{"8-9", 8, 9},
		{"0", 0, 0},
	}
	for _, tt := range tests {
		low, high := parseDIPBits(tt.input)
		if low != tt.wantLow || high != tt.wantHigh {
			t.Errorf("parseDIPBits(%q) = (%d,%d), want (%d,%d)", tt.input, low, high, tt.wantLow, tt.wantHigh)
		}
	}
}

func TestDIPSwitchCFGRoundTrip(t *testing.T) {
	// Simulate setting and reading a DIP switch value via CFG bits
	cfgData := make([]byte, 16)

	// Set "Free Play" (bit 10) to index 1 ("Off")
	SetBitRange(cfgData, 10, 10, 1)
	if val := GetBitRange(cfgData, 10, 10); val != 1 {
		t.Errorf("Free Play value = %d, want 1", val)
	}

	// Set "Lives" (bits 1-2) to index 2 ("5")
	SetBitRange(cfgData, 1, 2, 2)
	if val := GetBitRange(cfgData, 1, 2); val != 2 {
		t.Errorf("Lives value = %d, want 2", val)
	}

	// Set "Coin A" (bits 14-15) to index 3 ("2C/3P")
	SetBitRange(cfgData, 14, 15, 3)
	if val := GetBitRange(cfgData, 14, 15); val != 3 {
		t.Errorf("Coin A value = %d, want 3", val)
	}

	// Verify they didn't interfere with each other
	if val := GetBitRange(cfgData, 10, 10); val != 1 {
		t.Errorf("Free Play after other writes = %d, want 1", val)
	}
	if val := GetBitRange(cfgData, 1, 2); val != 2 {
		t.Errorf("Lives after other writes = %d, want 2", val)
	}
}

func TestParseMRAData_Empty(t *testing.T) {
	mra, err := ParseMRAData([]byte(`<misterromdescription></misterromdescription>`))
	if err != nil {
		t.Fatal(err)
	}
	if mra.SetName != "" {
		t.Errorf("expected empty setname, got %q", mra.SetName)
	}
	dips := ParseDIPSwitches(mra)
	if len(dips) != 0 {
		t.Errorf("expected no DIPs, got %d", len(dips))
	}
}
