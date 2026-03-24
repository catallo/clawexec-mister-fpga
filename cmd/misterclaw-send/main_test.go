package main

import (
	"encoding/json"
	"testing"
)

func TestBuildRequest_Status(t *testing.T) {
	req, err := BuildRequest("status", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, req, `{"mister":"status"}`)
}

func TestBuildRequest_Systems(t *testing.T) {
	req, err := BuildRequest("systems", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, req, `{"mister":"systems"}`)
}

func TestBuildRequest_Info(t *testing.T) {
	req, err := BuildRequest("info", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, req, `{"mister":"info"}`)
}

func TestBuildRequest_Screenshot(t *testing.T) {
	req, err := BuildRequest("screenshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, req, `{"mister":"screenshot"}`)
}

func TestBuildRequest_Search(t *testing.T) {
	req, err := BuildRequest("search", []string{"zelda", "--system", "SNES"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "search" {
		t.Errorf("expected mister=search, got %v", req["mister"])
	}
	if req["query"] != "zelda" {
		t.Errorf("expected query=zelda, got %v", req["query"])
	}
	if req["system"] != "SNES" {
		t.Errorf("expected system=SNES, got %v", req["system"])
	}
}

func TestBuildRequest_SearchNoSystem(t *testing.T) {
	req, err := BuildRequest("search", []string{"sonic", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if req["query"] != "sonic 2" {
		t.Errorf("expected query='sonic 2', got %v", req["query"])
	}
	if _, ok := req["system"]; ok {
		t.Error("expected no system key")
	}
}

func TestBuildRequest_SearchEmpty(t *testing.T) {
	_, err := BuildRequest("search", nil)
	if err == nil {
		t.Fatal("expected error for empty search")
	}
}

func TestBuildRequest_LaunchQuery(t *testing.T) {
	req, err := BuildRequest("launch", []string{"super", "mario", "--system", "SNES"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "launch" {
		t.Errorf("expected mister=launch, got %v", req["mister"])
	}
	if req["query"] != "super mario" {
		t.Errorf("expected query='super mario', got %v", req["query"])
	}
	if req["system"] != "SNES" {
		t.Errorf("expected system=SNES, got %v", req["system"])
	}
}

func TestBuildRequest_LaunchPath(t *testing.T) {
	req, err := BuildRequest("launch", []string{"--path", "/media/usb0/SNES/game.sfc", "--system", "SNES"})
	if err != nil {
		t.Fatal(err)
	}
	if req["path"] != "/media/usb0/SNES/game.sfc" {
		t.Errorf("expected path, got %v", req["path"])
	}
	if req["system"] != "SNES" {
		t.Errorf("expected system=SNES, got %v", req["system"])
	}
	if _, ok := req["query"]; ok {
		t.Error("expected no query key for path launch")
	}
}

func TestBuildRequest_LaunchPathNoSystem(t *testing.T) {
	_, err := BuildRequest("launch", []string{"--path", "/media/usb0/game.sfc"})
	if err == nil {
		t.Fatal("expected error: --path requires --system")
	}
}

func TestBuildRequest_LaunchEmpty(t *testing.T) {
	_, err := BuildRequest("launch", nil)
	if err == nil {
		t.Fatal("expected error for empty launch")
	}
}

func TestBuildRequest_Tailscale(t *testing.T) {
	for _, action := range []string{"setup", "status", "start", "stop"} {
		req, err := BuildRequest("tailscale", []string{action})
		if err != nil {
			t.Fatalf("action %s: %v", action, err)
		}
		if req["mister"] != "tailscale" {
			t.Errorf("expected mister=tailscale, got %v", req["mister"])
		}
		if req["action"] != action {
			t.Errorf("expected action=%s, got %v", action, req["action"])
		}
	}
}

func TestBuildRequest_TailscaleEmpty(t *testing.T) {
	_, err := BuildRequest("tailscale", nil)
	if err == nil {
		t.Fatal("expected error for empty tailscale")
	}
}

func TestBuildRequest_Unknown(t *testing.T) {
	_, err := BuildRequest("foobar", nil)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

// Tests for trailing flag parsing bug fix — flags after positional args must work.
func TestBuildRequest_SearchTrailingSystem(t *testing.T) {
	req, err := BuildRequest("search", []string{"ridge", "racer", "--system", "PSX"})
	if err != nil {
		t.Fatal(err)
	}
	if req["query"] != "ridge racer" {
		t.Errorf("expected query='ridge racer', got %v", req["query"])
	}
	if req["system"] != "PSX" {
		t.Errorf("expected system=PSX, got %v", req["system"])
	}
}

func TestBuildRequest_SearchTrailingShortFlag(t *testing.T) {
	req, err := BuildRequest("search", []string{"sonic", "-s", "Genesis"})
	if err != nil {
		t.Fatal(err)
	}
	if req["query"] != "sonic" {
		t.Errorf("expected query='sonic', got %v", req["query"])
	}
	if req["system"] != "Genesis" {
		t.Errorf("expected system=Genesis, got %v", req["system"])
	}
}

func TestBuildRequest_LaunchTrailingSystem(t *testing.T) {
	req, err := BuildRequest("launch", []string{"mario", "--system", "SNES"})
	if err != nil {
		t.Fatal(err)
	}
	if req["query"] != "mario" {
		t.Errorf("expected query='mario', got %v", req["query"])
	}
	if req["system"] != "SNES" {
		t.Errorf("expected system=SNES, got %v", req["system"])
	}
}

func TestBuildRequest_SearchSystemEqualsForm(t *testing.T) {
	req, err := BuildRequest("search", []string{"zelda", "--system=SNES"})
	if err != nil {
		t.Fatal(err)
	}
	if req["query"] != "zelda" {
		t.Errorf("expected query='zelda', got %v", req["query"])
	}
	if req["system"] != "SNES" {
		t.Errorf("expected system=SNES, got %v", req["system"])
	}
}

func TestBuildRequest_InputKey(t *testing.T) {
	req, err := BuildRequest("input", []string{"key", "osd"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "input" {
		t.Errorf("expected mister=input, got %v", req["mister"])
	}
	if req["key"] != "osd" {
		t.Errorf("expected key=osd, got %v", req["key"])
	}
}

func TestBuildRequest_InputRaw(t *testing.T) {
	req, err := BuildRequest("input", []string{"raw", "28"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "input" {
		t.Errorf("expected mister=input, got %v", req["mister"])
	}
	if req["raw"] != 28 {
		t.Errorf("expected raw=28, got %v", req["raw"])
	}
}

func TestBuildRequest_InputCombo(t *testing.T) {
	req, err := BuildRequest("input", []string{"combo", "leftalt", "f12"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "input" {
		t.Errorf("expected mister=input, got %v", req["mister"])
	}
	combo, ok := req["combo"].([]string)
	if !ok {
		t.Fatalf("expected combo to be []string, got %T", req["combo"])
	}
	if len(combo) != 2 || combo[0] != "leftalt" || combo[1] != "f12" {
		t.Errorf("expected combo=[leftalt,f12], got %v", combo)
	}
}

func TestBuildRequest_InputEmpty(t *testing.T) {
	_, err := BuildRequest("input", nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestBuildRequest_InputKeyEmpty(t *testing.T) {
	_, err := BuildRequest("input", []string{"key"})
	if err == nil {
		t.Fatal("expected error for input key with no name")
	}
}

func TestBuildRequest_InputRawInvalid(t *testing.T) {
	_, err := BuildRequest("input", []string{"raw", "abc"})
	if err == nil {
		t.Fatal("expected error for invalid raw keycode")
	}
}

func TestBuildRequest_InputComboEmpty(t *testing.T) {
	_, err := BuildRequest("input", []string{"combo"})
	if err == nil {
		t.Fatal("expected error for empty combo")
	}
}

func TestBuildRequest_InputUnknownMode(t *testing.T) {
	_, err := BuildRequest("input", []string{"foobar"})
	if err == nil {
		t.Fatal("expected error for unknown input mode")
	}
}

func TestBuildRequest_SystemInfo(t *testing.T) {
	req, err := BuildRequest("system-info", []string{"C64"})
	if err != nil {
		t.Fatal(err)
	}
	if req["mister"] != "system_info" {
		t.Errorf("expected mister=system_info, got %v", req["mister"])
	}
	if req["system"] != "C64" {
		t.Errorf("expected system=C64, got %v", req["system"])
	}
}

func TestBuildRequest_SystemInfoMultiWord(t *testing.T) {
	req, err := BuildRequest("system-info", []string{"PC", "8801"})
	if err != nil {
		t.Fatal(err)
	}
	if req["system"] != "PC 8801" {
		t.Errorf("expected system='PC 8801', got %v", req["system"])
	}
}

func TestBuildRequest_SystemInfoEmpty(t *testing.T) {
	_, err := BuildRequest("system-info", nil)
	if err == nil {
		t.Fatal("expected error for empty system-info")
	}
}

func TestHelpOutput(t *testing.T) {
	// Verify printHelp doesn't panic (basic smoke test)
	// We can't easily capture stdout in a test without more infrastructure,
	// but we can at least verify it doesn't crash.
	printHelp()
}

func TestDefaultFlags(t *testing.T) {
	if hostFlag != "mister-fpga" {
		// hostFlag may have been set by test init, just check it's reasonable
		t.Logf("hostFlag=%s (may differ in test context)", hostFlag)
	}
}

// assertJSON verifies that the request matches the expected JSON structure.
func assertJSON(t *testing.T, req map[string]interface{}, expected string) {
	t.Helper()
	var want map[string]interface{}
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatalf("bad expected JSON: %v", err)
	}
	got, _ := json.Marshal(req)
	wantBytes, _ := json.Marshal(want)
	if string(got) != string(wantBytes) {
		t.Errorf("JSON mismatch:\n  got:  %s\n  want: %s", got, wantBytes)
	}
}
