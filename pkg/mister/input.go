package mister

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bendahl/uinput"
)

// KeyboardDevice abstracts uinput.Keyboard for testing.
type KeyboardDevice interface {
	KeyPress(key int) error
	KeyDown(key int) error
	KeyUp(key int) error
	Close() error
}

// keyboardCreator is the function used to create a keyboard device.
// Override in tests to inject a mock.
var keyboardCreator = func() (KeyboardDevice, error) {
	return uinput.CreateKeyboard("/dev/uinput", []byte("misterclaw"))
}

var (
	kbMu   sync.Mutex
	kbInst KeyboardDevice
)

// getKeyboard returns the lazily-created shared keyboard instance.
func getKeyboard() (KeyboardDevice, error) {
	kbMu.Lock()
	defer kbMu.Unlock()
	if kbInst != nil {
		return kbInst, nil
	}
	kb, err := keyboardCreator()
	if err != nil {
		return nil, fmt.Errorf("creating keyboard device: %w", err)
	}
	kbInst = kb
	return kbInst, nil
}

// CloseKeyboard closes the shared keyboard device if open.
func CloseKeyboard() {
	kbMu.Lock()
	defer kbMu.Unlock()
	if kbInst != nil {
		kbInst.Close()
		kbInst = nil
	}
}

// KeyNames maps friendly names to uinput key codes.
// Includes all standard keys plus MiSTer-specific named actions.
var KeyNames = map[string]int{
	// Arrow keys
	"up":    uinput.KeyUp,
	"down":  uinput.KeyDown,
	"left":  uinput.KeyLeft,
	"right": uinput.KeyRight,

	// MiSTer named actions (single key)
	"confirm":           uinput.KeyEnter,
	"menu":              uinput.KeyEsc,
	"osd":               uinput.KeyF12,
	"pair_bluetooth":    uinput.KeyF11,
	"change_background": uinput.KeyF1,
	"toggle_core_dates": uinput.KeyF2,
	"console":           uinput.KeyF9,
	"exit_console":      uinput.KeyF12,
	"back":              uinput.KeyBackspace,
	"cancel":            uinput.KeyEsc,

	// Volume
	"volume_up":   uinput.KeyVolumeup,
	"volume_down": uinput.KeyVolumedown,
	"volume_mute": uinput.KeyMute,

	// Standard key names (for use in combos and raw access)
	"esc":         uinput.KeyEsc,
	"enter":       uinput.KeyEnter,
	"space":       uinput.KeySpace,
	"tab":         uinput.KeyTab,
	"backspace":   uinput.KeyBackspace,
	"delete":      uinput.KeyDelete,
	"insert":      uinput.KeyInsert,
	"home":        uinput.KeyHome,
	"end":         uinput.KeyEnd,
	"pageup":      uinput.KeyPageup,
	"pagedown":    uinput.KeyPagedown,
	"scrolllock":  uinput.KeyScrolllock,
	"pause":       uinput.KeyPause,
	"sysrq":       uinput.KeySysrq,

	// Function keys
	"f1":  uinput.KeyF1,
	"f2":  uinput.KeyF2,
	"f3":  uinput.KeyF3,
	"f4":  uinput.KeyF4,
	"f5":  uinput.KeyF5,
	"f6":  uinput.KeyF6,
	"f7":  uinput.KeyF7,
	"f8":  uinput.KeyF8,
	"f9":  uinput.KeyF9,
	"f10": uinput.KeyF10,
	"f11": uinput.KeyF11,
	"f12": uinput.KeyF12,

	// Modifiers
	"leftshift":  uinput.KeyLeftshift,
	"rightshift": uinput.KeyRightshift,
	"leftctrl":   uinput.KeyLeftctrl,
	"rightctrl":  uinput.KeyRightctrl,
	"leftalt":    uinput.KeyLeftalt,
	"rightalt":   uinput.KeyRightalt,
	"leftmeta":   uinput.KeyLeftmeta,
	"rightmeta":  uinput.KeyRightmeta,
	"win":        uinput.KeyLeftmeta,
}

// namedCombos maps MiSTer action names that require key combos.
var namedCombos = map[string][]int{
	"core_select":    {uinput.KeyLeftalt, uinput.KeyF12},
	"screenshot":     {uinput.KeyLeftalt, uinput.KeyScrolllock},
	"raw_screenshot": {uinput.KeyLeftalt, uinput.KeyLeftshift, uinput.KeyScrolllock},
	"user":           {uinput.KeyLeftctrl, uinput.KeyLeftalt, uinput.KeyRightalt},
	"reset":          {uinput.KeyLeftshift, uinput.KeyLeftctrl, uinput.KeyLeftalt, uinput.KeyRightalt},
	"computer_osd":   {uinput.KeyLeftmeta, uinput.KeyF12},
}

// PressKey presses a named key or named combo action.
func PressKey(name string) error {
	name = strings.ToLower(name)

	// Check if it's a named combo first
	if codes, ok := namedCombos[name]; ok {
		return pressCombo(codes)
	}

	code, ok := KeyNames[name]
	if !ok {
		return fmt.Errorf("unknown key name: %q", name)
	}

	kb, err := getKeyboard()
	if err != nil {
		return err
	}
	return kb.KeyPress(code)
}

// PressRawKey presses a key by its raw Linux keycode.
func PressRawKey(code int) error {
	kb, err := getKeyboard()
	if err != nil {
		return err
	}
	return kb.KeyPress(code)
}

// PressCombo presses a combination of keys by name (e.g. ["leftalt", "f12"]).
func PressCombo(names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("combo requires at least one key")
	}
	codes := make([]int, len(names))
	for i, name := range names {
		name = strings.ToLower(name)
		code, ok := KeyNames[name]
		if !ok {
			return fmt.Errorf("unknown key name in combo: %q", name)
		}
		codes[i] = code
	}
	return pressCombo(codes)
}

// pressCombo holds down all keys in order, then releases in reverse.
func pressCombo(codes []int) error {
	kb, err := getKeyboard()
	if err != nil {
		return err
	}

	// Press all keys down with 40ms delay between each
	for i, code := range codes {
		if err := kb.KeyDown(code); err != nil {
			// Release any already-pressed keys on error
			for j := i - 1; j >= 0; j-- {
				kb.KeyUp(codes[j])
			}
			return fmt.Errorf("key down %d: %w", code, err)
		}
		if i < len(codes)-1 {
			time.Sleep(40 * time.Millisecond)
		}
	}

	// Small delay before releasing
	time.Sleep(40 * time.Millisecond)

	// Release all keys in reverse order
	for i := len(codes) - 1; i >= 0; i-- {
		if err := kb.KeyUp(codes[i]); err != nil {
			return fmt.Errorf("key up %d: %w", codes[i], err)
		}
	}

	return nil
}
