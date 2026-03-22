package mister

import (
	"fmt"
	"sync"
	"testing"
)

// mockKeyboard records all key events for verification.
type mockKeyboard struct {
	mu     sync.Mutex
	events []keyEvent
	err    error // if set, all calls return this error
}

type keyEvent struct {
	action string // "press", "down", "up"
	code   int
}

func (m *mockKeyboard) KeyPress(key int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, keyEvent{"press", key})
	return nil
}

func (m *mockKeyboard) KeyDown(key int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, keyEvent{"down", key})
	return nil
}

func (m *mockKeyboard) KeyUp(key int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, keyEvent{"up", key})
	return nil
}

func (m *mockKeyboard) Close() error {
	return nil
}

func (m *mockKeyboard) getEvents() []keyEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]keyEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// withMockKeyboard installs a mock keyboard and returns it.
// Restores the original creator on cleanup.
func withMockKeyboard(t *testing.T) *mockKeyboard {
	t.Helper()
	mock := &mockKeyboard{}
	origCreator := keyboardCreator
	origInst := kbInst

	keyboardCreator = func() (KeyboardDevice, error) {
		return mock, nil
	}
	kbInst = nil // force re-creation

	t.Cleanup(func() {
		keyboardCreator = origCreator
		kbMu.Lock()
		kbInst = origInst
		kbMu.Unlock()
	})

	return mock
}

func TestPressKey_Named(t *testing.T) {
	mock := withMockKeyboard(t)

	if err := PressKey("osd"); err != nil {
		t.Fatalf("PressKey(osd): %v", err)
	}

	events := mock.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 1 event, got %d: %v", len(events), events)
	}
	if events[0].action != "down" {
		t.Errorf("expected down, got %s", events[0].action)
	}
	if events[0].code != KeyNames["f12"] {
		t.Errorf("expected F12 code %d, got %d", KeyNames["f12"], events[0].code)
	}
}

func TestPressKey_NamedCombo(t *testing.T) {
	mock := withMockKeyboard(t)

	if err := PressKey("core_select"); err != nil {
		t.Fatalf("PressKey(core_select): %v", err)
	}

	events := mock.getEvents()
	// core_select = leftalt + f12 → down, down, up, up
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d: %v", len(events), events)
	}
	if events[0].action != "down" || events[1].action != "down" {
		t.Error("expected two key-down events first")
	}
	if events[2].action != "up" || events[3].action != "up" {
		t.Error("expected two key-up events after")
	}
}

func TestPressKey_CaseInsensitive(t *testing.T) {
	mock := withMockKeyboard(t)

	if err := PressKey("OSD"); err != nil {
		t.Fatalf("PressKey(OSD): %v", err)
	}
	if len(mock.getEvents()) != 2 {
		t.Error("expected 2 events for case-insensitive key")
	}
}

func TestPressKey_Unknown(t *testing.T) {
	withMockKeyboard(t)

	err := PressKey("nonexistent_key_xyz")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestPressRawKey(t *testing.T) {
	mock := withMockKeyboard(t)

	if err := PressRawKey(28); err != nil {
		t.Fatalf("PressRawKey(28): %v", err)
	}

	events := mock.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].code != 28 {
		t.Errorf("expected code 28, got %d", events[0].code)
	}
}

func TestPressCombo(t *testing.T) {
	mock := withMockKeyboard(t)

	if err := PressCombo([]string{"leftalt", "f12"}); err != nil {
		t.Fatalf("PressCombo: %v", err)
	}

	events := mock.getEvents()
	if len(events) != 4 {
		t.Fatalf("expected 4 events (2 down + 2 up), got %d: %v", len(events), events)
	}
	// Down in order
	if events[0].action != "down" || events[0].code != KeyNames["leftalt"] {
		t.Errorf("event[0]: expected down leftalt, got %v", events[0])
	}
	if events[1].action != "down" || events[1].code != KeyNames["f12"] {
		t.Errorf("event[1]: expected down f12, got %v", events[1])
	}
	// Up in reverse
	if events[2].action != "up" || events[2].code != KeyNames["f12"] {
		t.Errorf("event[2]: expected up f12, got %v", events[2])
	}
	if events[3].action != "up" || events[3].code != KeyNames["leftalt"] {
		t.Errorf("event[3]: expected up leftalt, got %v", events[3])
	}
}

func TestPressCombo_Empty(t *testing.T) {
	withMockKeyboard(t)

	err := PressCombo(nil)
	if err == nil {
		t.Fatal("expected error for empty combo")
	}
}

func TestPressCombo_UnknownKey(t *testing.T) {
	withMockKeyboard(t)

	err := PressCombo([]string{"leftalt", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown key in combo")
	}
}

func TestKeyboardLazyInit(t *testing.T) {
	created := 0
	mock := &mockKeyboard{}

	origCreator := keyboardCreator
	origInst := kbInst

	keyboardCreator = func() (KeyboardDevice, error) {
		created++
		return mock, nil
	}
	kbMu.Lock()
	kbInst = nil
	kbMu.Unlock()

	t.Cleanup(func() {
		keyboardCreator = origCreator
		kbMu.Lock()
		kbInst = origInst
		kbMu.Unlock()
	})

	// Multiple calls should only create once
	PressKey("osd")
	PressKey("menu")
	PressRawKey(1)

	if created != 1 {
		t.Errorf("expected keyboard created once, got %d", created)
	}
}

func TestKeyboardCreationError(t *testing.T) {
	origCreator := keyboardCreator
	origInst := kbInst

	keyboardCreator = func() (KeyboardDevice, error) {
		return nil, fmt.Errorf("no /dev/uinput")
	}
	kbMu.Lock()
	kbInst = nil
	kbMu.Unlock()

	t.Cleanup(func() {
		keyboardCreator = origCreator
		kbMu.Lock()
		kbInst = origInst
		kbMu.Unlock()
	})

	err := PressKey("osd")
	if err == nil {
		t.Fatal("expected error when keyboard creation fails")
	}
}

func TestCloseKeyboard(t *testing.T) {
	mock := withMockKeyboard(t)

	// Use the keyboard to force creation
	PressKey("osd")
	_ = mock

	// Close should nil out the instance
	CloseKeyboard()

	kbMu.Lock()
	inst := kbInst
	kbMu.Unlock()

	if inst != nil {
		t.Error("expected kbInst to be nil after CloseKeyboard")
	}
}

func TestKeyNames_AllNamedKeys(t *testing.T) {
	// Verify all required named keys exist
	required := []string{
		"up", "down", "left", "right",
		"confirm", "menu", "osd", "pair_bluetooth",
		"console", "back",
		"volume_up", "volume_down", "volume_mute",
	}
	for _, name := range required {
		if _, ok := KeyNames[name]; !ok {
			t.Errorf("missing required key name: %s", name)
		}
	}
}

func TestNamedCombos_AllRequired(t *testing.T) {
	required := []string{
		"core_select", "screenshot", "user", "reset",
	}
	for _, name := range required {
		if _, ok := namedCombos[name]; !ok {
			t.Errorf("missing required named combo: %s", name)
		}
	}
}

func TestPressCombo_KeyDownError(t *testing.T) {
	// Test that partial key-down errors release already-pressed keys
	callCount := 0
	mock := &mockKeyboard{}
	origErr := mock.err

	origCreator := keyboardCreator
	origInst := kbInst

	// Create a keyboard that fails on second KeyDown
	failOnSecond := &failingKeyboard{failAt: 1}
	keyboardCreator = func() (KeyboardDevice, error) {
		callCount++
		return failOnSecond, nil
	}
	kbMu.Lock()
	kbInst = nil
	kbMu.Unlock()

	t.Cleanup(func() {
		keyboardCreator = origCreator
		kbMu.Lock()
		kbInst = origInst
		kbMu.Unlock()
		mock.err = origErr
	})

	err := PressCombo([]string{"leftalt", "f12"})
	if err == nil {
		t.Fatal("expected error from failing KeyDown")
	}

	// Should have released the first key
	if len(failOnSecond.upCalls) == 0 {
		t.Error("expected KeyUp to be called for cleanup")
	}
}

// failingKeyboard fails KeyDown on the Nth call.
type failingKeyboard struct {
	downCount int
	failAt    int
	upCalls   []int
}

func (f *failingKeyboard) KeyPress(key int) error { return nil }

func (f *failingKeyboard) KeyDown(key int) error {
	if f.downCount == f.failAt {
		f.downCount++
		return fmt.Errorf("simulated failure")
	}
	f.downCount++
	return nil
}

func (f *failingKeyboard) KeyUp(key int) error {
	f.upCalls = append(f.upCalls, key)
	return nil
}

func (f *failingKeyboard) Close() error { return nil }
