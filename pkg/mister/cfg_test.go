package mister

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSetBit(t *testing.T) {
	data := make([]byte, 4) // 32 bits

	// All bits start as 0
	for i := 0; i < 32; i++ {
		if GetBit(data, i) {
			t.Errorf("bit %d should be 0 initially", i)
		}
	}

	// Set some bits
	SetBit(data, 0, true)
	SetBit(data, 7, true)
	SetBit(data, 8, true)
	SetBit(data, 31, true)

	if !GetBit(data, 0) {
		t.Error("bit 0 should be set")
	}
	if !GetBit(data, 7) {
		t.Error("bit 7 should be set")
	}
	if !GetBit(data, 8) {
		t.Error("bit 8 should be set")
	}
	if !GetBit(data, 31) {
		t.Error("bit 31 should be set")
	}
	if GetBit(data, 1) {
		t.Error("bit 1 should not be set")
	}

	// Clear a bit
	SetBit(data, 7, false)
	if GetBit(data, 7) {
		t.Error("bit 7 should be cleared")
	}

	// Verify byte values
	if data[0] != 0x01 { // bit 0
		t.Errorf("byte 0 = %02x, want 0x01", data[0])
	}
	if data[1] != 0x01 { // bit 8
		t.Errorf("byte 1 = %02x, want 0x01", data[1])
	}
	if data[3] != 0x80 { // bit 31
		t.Errorf("byte 3 = %02x, want 0x80", data[3])
	}
}

func TestGetSetBit_OutOfRange(t *testing.T) {
	data := make([]byte, 2)

	// Getting out of range returns false
	if GetBit(data, 100) {
		t.Error("out of range bit should return false")
	}

	// Setting out of range is a no-op
	SetBit(data, 100, true)
	if data[0] != 0 || data[1] != 0 {
		t.Error("out of range set should not modify data")
	}
}

func TestGetBitRange(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		start    int
		end      int
		expected int
	}{
		{"single bit 0", []byte{0x01}, 0, 0, 1},
		{"single bit 1", []byte{0x02}, 1, 1, 1},
		{"bits 0-1 = 3", []byte{0x03}, 0, 1, 3},
		{"bits 0-1 = 2", []byte{0x02}, 0, 1, 2},
		{"bits 4-5 = 1", []byte{0x10}, 4, 5, 1},
		{"bits 4-5 = 3", []byte{0x30}, 4, 5, 3},
		{"bits 8-9 across bytes", []byte{0x00, 0x03}, 8, 9, 3},
		{"bits 8-9 = 2", []byte{0x00, 0x02}, 8, 9, 2},
		{"all zeros", []byte{0x00, 0x00}, 0, 7, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBitRange(tt.data, tt.start, tt.end)
			if got != tt.expected {
				t.Errorf("GetBitRange(%v, %d, %d) = %d, want %d", tt.data, tt.start, tt.end, got, tt.expected)
			}
		})
	}
}

func TestSetBitRange(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		end      int
		value    int
		expected int
	}{
		{"set 2 bits to 3", 0, 1, 3, 3},
		{"set 2 bits to 1", 0, 1, 1, 1},
		{"set 2 bits to 2", 0, 1, 2, 2},
		{"set bits 4-5 to 2", 4, 5, 2, 2},
		{"set bits 8-9 to 3", 8, 9, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 4)
			SetBitRange(data, tt.start, tt.end, tt.value)
			got := GetBitRange(data, tt.start, tt.end)
			if got != tt.expected {
				t.Errorf("after SetBitRange(%d, %d, %d): GetBitRange = %d, want %d", tt.start, tt.end, tt.value, got, tt.expected)
			}
		})
	}
}

func TestSetBitRange_RoundTrip(t *testing.T) {
	data := make([]byte, 16)

	// Set multiple options at different bit positions
	SetBitRange(data, 4, 6, 5)  // bits 4-6 = 5
	SetBitRange(data, 8, 9, 2)  // bits 8-9 = 2
	SetBitRange(data, 13, 14, 1) // bits 13-14 = 1

	if got := GetBitRange(data, 4, 6); got != 5 {
		t.Errorf("bits 4-6 = %d, want 5", got)
	}
	if got := GetBitRange(data, 8, 9); got != 2 {
		t.Errorf("bits 8-9 = %d, want 2", got)
	}
	if got := GetBitRange(data, 13, 14); got != 1 {
		t.Errorf("bits 13-14 = %d, want 1", got)
	}
}

func TestBackupCFG(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.CFG")

	// Create a CFG file
	original := []byte{0x01, 0x02, 0x03, 0x04}
	if err := os.WriteFile(cfgPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	// Backup
	if err := BackupCFG(cfgPath); err != nil {
		t.Fatal(err)
	}

	// Find the backup file
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var bakFile string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.CFG.bak-") {
			bakFile = e.Name()
			break
		}
	}

	if bakFile == "" {
		t.Fatal("backup file not found")
	}

	// Verify backup has correct timestamp format
	if !strings.Contains(bakFile, ".bak-") {
		t.Errorf("backup filename %s doesn't contain .bak-", bakFile)
	}

	// Verify backup contents match
	bakData, err := os.ReadFile(filepath.Join(tmpDir, bakFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(bakData) != len(original) {
		t.Errorf("backup size %d != original %d", len(bakData), len(original))
	}
	for i := range original {
		if bakData[i] != original[i] {
			t.Errorf("backup byte %d = %02x, want %02x", i, bakData[i], original[i])
		}
	}
}

func TestWriteCFG_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.CFG")

	// Create initial CFG
	if err := os.WriteFile(cfgPath, []byte{0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}

	// Write new CFG (should create backup first)
	newData := []byte{0xFF, 0xFF}
	if err := WriteCFG(cfgPath, newData); err != nil {
		t.Fatal(err)
	}

	// Verify new data was written
	got, err := ReadCFG(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0xFF || got[1] != 0xFF {
		t.Errorf("written data = %v, want [0xFF, 0xFF]", got)
	}

	// Verify backup exists
	entries, _ := os.ReadDir(tmpDir)
	bakFound := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.CFG.bak-") {
			bakFound = true
			break
		}
	}
	if !bakFound {
		t.Error("WriteCFG should create a backup file")
	}
}

func TestWriteCFG_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "new.CFG")

	// Writing to a new file (no backup needed, should not error)
	if err := WriteCFG(cfgPath, []byte{0x01}); err != nil {
		t.Fatalf("WriteCFG to new file should succeed: %v", err)
	}

	got, err := ReadCFG(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 0x01 {
		t.Errorf("data = %v, want [0x01]", got)
	}
}

func TestVisibility_HideWhenBitSet(t *testing.T) {
	// H1O34,Hidden Opt,A,B — hide when bit 1 is set
	items := ParseConfStr("CORE;;H1O34,Aspect,Wide,Narrow")

	var item *MenuItem
	for i := range items {
		if items[i].Name == "Aspect" {
			item = &items[i]
			break
		}
	}
	if item == nil {
		t.Fatal("expected to find Aspect option")
	}
	if len(item.HideConditions) == 0 {
		t.Fatal("expected hide conditions")
	}

	// With bit 1 = 0, should be visible
	cfgZero := make([]byte, 16)
	if !item.Visible(cfgZero) {
		t.Error("item should be visible when hide-bit is 0")
	}

	// With bit 1 = 1, should be hidden
	cfgSet := make([]byte, 16)
	SetBit(cfgSet, 1, true)
	if item.Visible(cfgSet) {
		t.Error("item should be hidden when hide-bit is 1")
	}
}

func TestVisibility_HideWhenBitClear(t *testing.T) {
	// h1O34,Aspect,Wide,Narrow — hide when bit 1 is 0 (inverted)
	items := ParseConfStr("CORE;;h1O34,Aspect,Wide,Narrow")

	var item *MenuItem
	for i := range items {
		if items[i].Name == "Aspect" {
			item = &items[i]
			break
		}
	}
	if item == nil {
		t.Fatal("expected to find Aspect option")
	}

	// With bit 1 = 0, should be hidden (inverted: hide when 0)
	cfgZero := make([]byte, 16)
	if item.Visible(cfgZero) {
		t.Error("item should be hidden when hide-bit is 0 (inverted)")
	}

	// With bit 1 = 1, should be visible
	cfgSet := make([]byte, 16)
	SetBit(cfgSet, 1, true)
	if !item.Visible(cfgSet) {
		t.Error("item should be visible when hide-bit is 1 (inverted)")
	}
}

func TestVisibleMenu(t *testing.T) {
	// Simulate a core with some hidden items
	items := ParseConfStr("CORE;;O12,Volume,Low,High;H1O34,Aspect,Wide,Narrow;O56,Region,US,JP")
	core := &CoreOSD{
		CoreName: "TestCore",
		Menu:     items,
	}

	// All zeros — H1 means "hide when bit 1 = 1", so with bit 1 = 0, everything visible
	cfgZero := make([]byte, 16)
	visible := VisibleMenu(core, cfgZero)
	// All items should be visible
	visibleNames := []string{}
	for _, v := range visible {
		if v.Name != "" {
			visibleNames = append(visibleNames, v.Name)
		}
	}

	if !containsStr2(visibleNames, "Volume") || !containsStr2(visibleNames, "Aspect") || !containsStr2(visibleNames, "Region") {
		t.Errorf("with all zeros, all items should be visible, got: %v", visibleNames)
	}

	// Set bit 1 — Aspect should be hidden
	cfgSet := make([]byte, 16)
	SetBit(cfgSet, 1, true)
	visible = VisibleMenu(core, cfgSet)
	visibleNames = []string{}
	for _, v := range visible {
		if v.Name != "" {
			visibleNames = append(visibleNames, v.Name)
		}
	}

	if !containsStr2(visibleNames, "Volume") || !containsStr2(visibleNames, "Region") {
		t.Error("Volume and Region should be visible")
	}
	if containsStr2(visibleNames, "Aspect") {
		t.Error("Aspect should be hidden when bit 1 is set")
	}
}

func TestTaitoSJ_AspectHidden(t *testing.T) {
	// Real-world Taito SJ CONF_STR fragment:
	// h0O34,Aspect Ratio,...  — hide when bit 0 = 0
	// With default CFG (all zeros), bit 0 = 0, so Aspect ratio should be HIDDEN
	items := ParseConfStr("TaitoSJ;;h0O34,Aspect ratio,Original,Full Screen,[ARC1],[ARC2];O5,Orientation,Horz,Vert")
	core := &CoreOSD{
		CoreName: "TaitoSJ",
		Menu:     items,
	}

	cfgDefault := make([]byte, 16) // all zeros
	visible := VisibleMenu(core, cfgDefault)

	for _, v := range visible {
		if v.Name == "Aspect ratio" {
			t.Error("Aspect ratio should be HIDDEN with default CFG (h0: hide when bit 0 = 0)")
		}
	}

	// Orientation should be visible (no hide conditions)
	found := false
	for _, v := range visible {
		if v.Name == "Orientation" {
			found = true
		}
	}
	if !found {
		t.Error("Orientation should be visible")
	}

	// Now set bit 0 — Aspect ratio should become visible
	cfgSet := make([]byte, 16)
	SetBit(cfgSet, 0, true)
	visible = VisibleMenu(core, cfgSet)

	found = false
	for _, v := range visible {
		if v.Name == "Aspect ratio" {
			found = true
		}
	}
	if !found {
		t.Error("Aspect ratio should be visible when bit 0 is set")
	}
}

func TestFindOption(t *testing.T) {
	items := ParseConfStr("CORE;;O12,Volume,Low,High;O34,Aspect Ratio,Wide,Narrow")
	core := &CoreOSD{CoreName: "Test", Menu: items}

	opt := FindOption(core, "Volume")
	if opt == nil || opt.Name != "Volume" {
		t.Error("should find Volume option")
	}

	opt = FindOption(core, "volume") // case-insensitive
	if opt == nil {
		t.Error("should find Volume case-insensitively")
	}

	opt = FindOption(core, "nonexistent")
	if opt != nil {
		t.Error("should return nil for nonexistent option")
	}
}

func TestFindOptionValue(t *testing.T) {
	items := ParseConfStr("CORE;;O12,Region,US,JP,EU")
	var opt *MenuItem
	for i := range items {
		if items[i].Name == "Region" {
			opt = &items[i]
			break
		}
	}
	if opt == nil {
		t.Fatal("expected Region option")
	}

	if idx := FindOptionValue(opt, "US"); idx != 0 {
		t.Errorf("US should be index 0, got %d", idx)
	}
	if idx := FindOptionValue(opt, "jp"); idx != 1 { // case-insensitive
		t.Errorf("JP should be index 1, got %d", idx)
	}
	if idx := FindOptionValue(opt, "EU"); idx != 2 {
		t.Errorf("EU should be index 2, got %d", idx)
	}
	if idx := FindOptionValue(opt, "ZZ"); idx != -1 {
		t.Errorf("ZZ should be -1, got %d", idx)
	}
}

func TestLetterToBit(t *testing.T) {
	tests := []struct {
		input byte
		want  int
	}{
		{'A', 0},
		{'B', 1},
		{'Z', 25},
		{'a', 32},
		{'b', 33},
		{'z', 57},
		{'0', 0},
		{'9', 9},
	}
	for _, tt := range tests {
		got := LetterToBit(tt.input)
		if got != tt.want {
			t.Errorf("LetterToBit(%c) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func containsStr2(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
