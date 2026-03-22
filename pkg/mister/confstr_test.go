package mister

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestParseConfStr_SNES(t *testing.T) {
	raw := "SNES;;" +
		"FS0,SFCSMCBS ,Load;" +
		"FC0,IPS,Load Cheat;" +
		"-;" +
		"O89,Aspect Ratio,Original,Full Screen,Custom;" +
		"O46,Scandoubler Fx,None,HQ2x,CRT 25%,CRT 50%,CRT 75%;" +
		"OAB,Video Mode,NTSC,PAL;" +
		"T0,Reset;" +
		"V,v1.0"

	items := ParseConfStr(raw)
	if len(items) == 0 {
		t.Fatal("expected items, got none")
	}

	// First item: core name label
	if items[0].Type != "label" || items[0].Name != "SNES" {
		t.Errorf("expected label SNES, got %s %s", items[0].Type, items[0].Name)
	}

	// File load
	found := false
	for _, item := range items {
		if item.Type == "file_load" && item.Label == "Load" {
			found = true
			if len(item.Extensions) < 2 {
				t.Errorf("expected at least 2 extensions, got %v", item.Extensions)
			}
			break
		}
	}
	if !found {
		t.Error("expected file_load item with label Load")
	}

	// Option: Aspect Ratio
	found = false
	for _, item := range items {
		if item.Type == "option" && item.Name == "Aspect Ratio" {
			found = true
			if item.Bit != 8 || item.BitHigh != 9 {
				t.Errorf("expected bits 8-9, got %d-%d", item.Bit, item.BitHigh)
			}
			if len(item.Values) != 3 {
				t.Errorf("expected 3 values, got %d: %v", len(item.Values), item.Values)
			}
			break
		}
	}
	if !found {
		t.Error("expected option Aspect Ratio")
	}

	// Trigger: Reset
	found = false
	for _, item := range items {
		if item.Type == "trigger" && item.Name == "Reset" {
			found = true
			if item.Bit != 0 {
				t.Errorf("expected bit 0, got %d", item.Bit)
			}
			break
		}
	}
	if !found {
		t.Error("expected trigger Reset")
	}

	// Version
	found = false
	for _, item := range items {
		if item.Type == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected version item")
	}
}

func TestParseConfStr_Separator(t *testing.T) {
	items := ParseConfStr("CORE;;-;T0,Reset")
	hasSep := false
	for _, item := range items {
		if item.Type == "separator" {
			hasSep = true
		}
	}
	if !hasSep {
		t.Error("expected separator")
	}
}

func TestParseConfStr_DIP(t *testing.T) {
	items := ParseConfStr("CORE;;DIP;T0,Reset")
	hasDIP := false
	for _, item := range items {
		if item.Type == "dip" {
			hasDIP = true
		}
	}
	if !hasDIP {
		t.Error("expected DIP item")
	}
}

func TestParseConfStr_SubPage(t *testing.T) {
	items := ParseConfStr("CORE;;P1,Audio Settings;O12,Volume,Low,High;P1O34,Bass,Off,On")
	var subPage *MenuItem
	for i := range items {
		if items[i].Type == "sub_page" {
			subPage = &items[i]
			break
		}
	}
	if subPage == nil {
		t.Fatal("expected sub_page item")
	}
	if subPage.PageID != 1 || subPage.Name != "Audio Settings" {
		t.Errorf("expected page 1 Audio Settings, got page %d %s", subPage.PageID, subPage.Name)
	}
}

func TestParseConfStr_FileLoadCore(t *testing.T) {
	items := ParseConfStr("CORE;;FC0,BIN,Load ROM")
	var fc *MenuItem
	for i := range items {
		if items[i].Type == "file_load_core" {
			fc = &items[i]
			break
		}
	}
	if fc == nil {
		t.Fatal("expected file_load_core item")
	}
	if fc.Label != "Load ROM" {
		t.Errorf("expected label 'Load ROM', got '%s'", fc.Label)
	}
	if len(fc.Extensions) == 0 || fc.Extensions[0] != "BIN" {
		t.Errorf("expected extension BIN, got %v", fc.Extensions)
	}
}

func TestParseConfStr_Mount(t *testing.T) {
	items := ParseConfStr("CORE;;S0,VHD,Mount VHD")
	var mount *MenuItem
	for i := range items {
		if items[i].Type == "mount" {
			mount = &items[i]
			break
		}
	}
	if mount == nil {
		t.Fatal("expected mount item")
	}
	if mount.Label != "Mount VHD" {
		t.Errorf("expected label 'Mount VHD', got '%s'", mount.Label)
	}
}

func TestParseConfStr_HideDisable(t *testing.T) {
	items := ParseConfStr("CORE;;H1O34,Hidden Opt,A,B;D2O56,Disabled Opt,C,D")
	var hide, disable *MenuItem
	for i := range items {
		if items[i].Type == "hide" {
			hide = &items[i]
		}
		if items[i].Type == "disable" {
			disable = &items[i]
		}
	}
	if hide == nil {
		t.Fatal("expected hide item")
	}
	if hide.Bit != 1 {
		t.Errorf("expected hide bit 1, got %d", hide.Bit)
	}
	if disable == nil {
		t.Fatal("expected disable item")
	}
	if disable.Bit != 2 {
		t.Errorf("expected disable bit 2, got %d", disable.Bit)
	}
}

func TestParseConfStr_Joystick(t *testing.T) {
	items := ParseConfStr("CORE;;J,A,B,X,Y,L,R,Select,Start")
	var joy *MenuItem
	for i := range items {
		if items[i].Type == "joystick" {
			joy = &items[i]
			break
		}
	}
	if joy == nil {
		t.Fatal("expected joystick item")
	}
	if len(joy.Values) < 8 {
		t.Errorf("expected at least 8 joystick values, got %d: %v", len(joy.Values), joy.Values)
	}
}

func TestParseConfStr_Info(t *testing.T) {
	items := ParseConfStr("CORE;;IVersion 1.0 by Author")
	var info *MenuItem
	for i := range items {
		if items[i].Type == "info" {
			info = &items[i]
			break
		}
	}
	if info == nil {
		t.Fatal("expected info item")
	}
	if info.Name != "Version 1.0 by Author" {
		t.Errorf("unexpected info name: %s", info.Name)
	}
}

func TestParseBitRange(t *testing.T) {
	tests := []struct {
		input    string
		wantLow  int
		wantHigh int
	}{
		{"0", 0, 0},
		{"3", 3, 3},
		{"89", 8, 9},
		{"AB", 10, 11},
		{"9A", 9, 10},
		{"AV", 10, 31},
		{"", 0, 0},
	}
	for _, tt := range tests {
		low, high := parseBitRange(tt.input)
		if low != tt.wantLow || high != tt.wantHigh {
			t.Errorf("parseBitRange(%q) = (%d,%d), want (%d,%d)", tt.input, low, high, tt.wantLow, tt.wantHigh)
		}
	}
}

func TestParseExtensions(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"BIN", []string{"BIN"}},
		{"BINSFC", []string{"BIN", "SFC"}},
		{"SFCSMCBS", []string{"SFC", "SMC", "BS"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseExtensions(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseExtensions(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseExtensions(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractConfStr_BraceFormat(t *testing.T) {
	sv := `
module top(
    input clk
);

localparam CONF_STR = {
    "SNES;;",
    "FS0,SFCSMCBS ,Load;",
    "O89,Aspect Ratio,Original,Full Screen;",
    "T0,Reset;"
};

endmodule
`
	got := ExtractConfStr(sv)
	if got == "" {
		t.Fatal("expected non-empty CONF_STR")
	}
	if !containsStr(got, "SNES") {
		t.Errorf("expected SNES in result, got: %s", got)
	}
	if !containsStr(got, "Aspect Ratio") {
		t.Errorf("expected Aspect Ratio in result, got: %s", got)
	}
}

func TestExtractConfStr_PlainString(t *testing.T) {
	sv := `parameter CONF_STR = "CORE;;O1,Aspect,4:3,16:9;T0,Reset;";`
	got := ExtractConfStr(sv)
	if got == "" {
		t.Fatal("expected non-empty CONF_STR")
	}
	if !containsStr(got, "CORE") {
		t.Errorf("expected CORE in result, got: %s", got)
	}
}

func TestExtractConfStr_NotFound(t *testing.T) {
	sv := `module top(); endmodule`
	got := ExtractConfStr(sv)
	if got != "" {
		t.Errorf("expected empty result, got: %s", got)
	}
}

func TestExtractCoreNameFromConfStr(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"SNES;;O1,Foo,A,B", "SNES"},
		{"MegaDrive;", "MegaDrive"},
		{"NES", "NES"},
		{"", ""},
	}
	for _, tt := range tests {
		got := ExtractCoreName(tt.raw)
		if got != tt.want {
			t.Errorf("ExtractCoreName(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestRepoToCoreName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SNES_MiSTer", "SNES"},
		{"Arcade-DonkeyKong_MiSTer", "DonkeyKong"},
		{"MegaDrive_MiSTer", "MegaDrive"},
		{"jtcps1", "jtcps1"},
	}
	for _, tt := range tests {
		got := RepoToCoreName(tt.input)
		if got != tt.want {
			t.Errorf("RepoToCoreName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLookupCoreOSD(t *testing.T) {
	db := &ConfStrDB{
		Cores: []CoreOSD{
			{CoreName: "SNES", Repo: "MiSTer-devel/SNES_MiSTer"},
			{CoreName: "Genesis", Repo: "MiSTer-devel/MegaDrive_MiSTer"},
		},
	}

	// By core name
	if got := LookupCoreOSD(db, "snes"); got == nil || got.CoreName != "SNES" {
		t.Error("expected to find SNES by name")
	}

	// By repo suffix
	if got := LookupCoreOSD(db, "megadrive"); got == nil || got.CoreName != "Genesis" {
		t.Error("expected to find Genesis by repo name megadrive")
	}

	// Not found
	if got := LookupCoreOSD(db, "nonexistent"); got != nil {
		t.Error("expected nil for nonexistent core")
	}
}

func TestLoadConfStrDB(t *testing.T) {
	// Create a temp DB file
	db := ConfStrDB{
		Version: "test",
		Cores: []CoreOSD{
			{
				CoreName:   "TestCore",
				Repo:       "test/TestCore_MiSTer",
				ConfStrRaw: "TestCore;;T0,Reset",
				Menu: []MenuItem{
					{Type: "label", Name: "TestCore"},
					{Type: "trigger", Bit: 0, Name: "Reset"},
				},
			},
		},
	}
	data, _ := json.Marshal(db)
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "confstr_db.json")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Override path and reset the once
	old := confstrDBPath
	confstrDBPath = tmpFile
	confstrDBOnce = syncOnce()
	defer func() {
		confstrDBPath = old
	}()

	loaded, err := LoadConfStrDB()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "test" {
		t.Errorf("expected version test, got %s", loaded.Version)
	}
	if len(loaded.Cores) != 1 {
		t.Errorf("expected 1 core, got %d", len(loaded.Cores))
	}
}

func TestParseConfStr_RealGenesis(t *testing.T) {
	// Real-world CONF_STR from Genesis/MegaDrive core
	raw := "GENESIS;ACTIVE;" +
		"FS0,BINGENMD ,Load ROM;" +
		"O67,Region,Auto,EU,JP,US;" +
		"ODE,Aspect Ratio,Original,Full Screen,Custom;" +
		"O23,Scandoubler Fx,None,HQ2x,CRT 25%,CRT 50%;" +
		"-;" +
		"P1,Audio Settings;" +
		"P1O4,FM Chip,YM2612,YM3438;" +
		"P1O5,Audio Filter,On,Off;" +
		"-;" +
		"R0,Reset;" +
		"J1,A,B,C,X,Y,Z,Mode,Start;" +
		"V,v1.0"

	items := ParseConfStr(raw)

	// Check we got a reasonable number of items
	if len(items) < 10 {
		t.Fatalf("expected at least 10 items, got %d", len(items))
	}

	// Check region option with letter bits
	var region *MenuItem
	for i := range items {
		if items[i].Type == "option" && items[i].Name == "Region" {
			region = &items[i]
			break
		}
	}
	if region == nil {
		t.Fatal("expected Region option")
	}
	if region.Bit != 6 || region.BitHigh != 7 {
		t.Errorf("expected bits 6-7, got %d-%d", region.Bit, region.BitHigh)
	}
	if len(region.Values) != 4 {
		t.Errorf("expected 4 region values, got %d", len(region.Values))
	}

	// Check aspect ratio with letter bits (D,E = 13,14)
	var aspect *MenuItem
	for i := range items {
		if items[i].Type == "option" && items[i].Name == "Aspect Ratio" {
			aspect = &items[i]
			break
		}
	}
	if aspect == nil {
		t.Fatal("expected Aspect Ratio option")
	}
	if aspect.Bit != 13 || aspect.BitHigh != 14 {
		t.Errorf("expected bits 13-14 (D,E), got %d-%d", aspect.Bit, aspect.BitHigh)
	}

	// Check reset
	var reset *MenuItem
	for i := range items {
		if items[i].Type == "reset" {
			reset = &items[i]
			break
		}
	}
	if reset == nil {
		t.Fatal("expected reset item")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findStr(s, sub))
}

func findStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// syncOnce returns a fresh sync.Once (for testing).
func syncOnce() sync.Once { return sync.Once{} }
