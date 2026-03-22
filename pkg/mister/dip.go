package mister

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DIPSize    = 8 // DIP switch data is 8 bytes (64 bits)
	DIPDirPath = "/media/fat/config/dips"
)

// DIPPath returns the .dip file path for a given MRA filename.
// MiSTer stores DIP files as /media/fat/config/dips/<name>.dip
// where <name> is the MRA filename without extension.
func DIPPath(mraFilename string) string {
	base := filepath.Base(mraFilename)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(DIPDirPath, name+".dip")
}

// ReadDIP reads a raw DIP file from disk.
func ReadDIP(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading DIP %s: %w", path, err)
	}
	return data, nil
}

// WriteDIP writes DIP data to disk, creating a timestamped backup first.
// Also ensures the dips directory exists.
func WriteDIP(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating dips directory %s: %w", dir, err)
	}

	// Backup before writing (if file exists)
	if err := BackupCFG(path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("backup before write: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing DIP %s: %w", path, err)
	}
	return nil
}

// ParseMRADefaults parses the switches default attribute into bytes.
// Format: "FF,FF,FF" or "00,7F,00,FF,00,00,00,00" (comma-separated hex bytes).
// Returns a slice padded/truncated to DIPSize bytes.
func ParseMRADefaults(defaultStr string) []byte {
	result := make([]byte, DIPSize)
	if defaultStr == "" {
		return result
	}

	parts := strings.Split(defaultStr, ",")
	for i, p := range parts {
		if i >= DIPSize {
			break
		}
		p = strings.TrimSpace(p)
		if b, err := hex.DecodeString(p); err == nil && len(b) == 1 {
			result[i] = b[0]
		}
	}
	return result
}

// GetMRADefaults returns the default DIP switch bytes from an MRA.
// If the MRA has no switches block, returns 8 zero bytes.
func GetMRADefaults(mra *MRA) []byte {
	for _, sw := range mra.Switches {
		if sw.Default != "" {
			return ParseMRADefaults(sw.Default)
		}
	}
	return make([]byte, DIPSize)
}

// LoadDIPData loads the current DIP switch state for an MRA.
// If no .dip file exists, returns the MRA defaults.
func LoadDIPData(dipPath string, mra *MRA) []byte {
	data, err := ReadDIP(dipPath)
	if err == nil && len(data) >= DIPSize {
		return data[:DIPSize]
	}
	// No .dip file — use MRA defaults
	return GetMRADefaults(mra)
}
