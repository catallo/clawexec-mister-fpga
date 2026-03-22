package mister

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ReadCFG reads a raw CFG file from disk.
func ReadCFG(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading CFG %s: %w", path, err)
	}
	return data, nil
}

// WriteCFG writes CFG data to disk, creating a timestamped backup first.
func WriteCFG(path string, data []byte) error {
	// Always backup before writing
	if err := BackupCFG(path); err != nil {
		// If the file doesn't exist yet, no backup needed
		if !os.IsNotExist(err) {
			return fmt.Errorf("backup before write: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing CFG %s: %w", path, err)
	}
	return nil
}

// BackupCFG copies a CFG file to path.bak-YYYYMMDD-HHMMSS.
func BackupCFG(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	ts := time.Now().Format("20060102-150405")
	bakPath := path + ".bak-" + ts

	dst, err := os.Create(bakPath)
	if err != nil {
		return fmt.Errorf("creating backup %s: %w", bakPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying to backup: %w", err)
	}
	return nil
}

// GetBit returns true if the specified bit is set in data.
// Bit 0 is the LSB of byte 0.
func GetBit(data []byte, bit int) bool {
	byteIdx := bit / 8
	bitIdx := uint(bit % 8)
	if byteIdx >= len(data) {
		return false
	}
	return data[byteIdx]&(1<<bitIdx) != 0
}

// SetBit sets or clears a single bit in data.
func SetBit(data []byte, bit int, value bool) {
	byteIdx := bit / 8
	bitIdx := uint(bit % 8)
	if byteIdx >= len(data) {
		return
	}
	if value {
		data[byteIdx] |= 1 << bitIdx
	} else {
		data[byteIdx] &^= 1 << bitIdx
	}
}

// GetBitRange reads a multi-bit value from data.
// Bits are numbered LSB-first: startBit is the lowest bit, endBit is the highest.
func GetBitRange(data []byte, startBit, endBit int) int {
	if startBit > endBit {
		startBit, endBit = endBit, startBit
	}
	val := 0
	for i := startBit; i <= endBit; i++ {
		if GetBit(data, i) {
			val |= 1 << uint(i-startBit)
		}
	}
	return val
}

// SetBitRange writes a multi-bit value into data.
func SetBitRange(data []byte, startBit, endBit int, value int) {
	if startBit > endBit {
		startBit, endBit = endBit, startBit
	}
	for i := startBit; i <= endBit; i++ {
		bitVal := value & (1 << uint(i-startBit))
		SetBit(data, i, bitVal != 0)
	}
}

// CFGPath returns the expected CFG file path for a given game/core name.
// MiSTer stores CFG files in /media/fat/config/<name>.CFG
func CFGPath(name string) string {
	return "/media/fat/config/" + name + ".CFG"
}
