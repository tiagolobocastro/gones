package gones

import (
	"fmt"
	"unsafe"
)

// "NES" + EOF
const NESMagicConstant = 0x1A53454E

type iNESFormat int

const (
	iNESInvalid = iota
	iNES0       // Archaic iNES format
	iNES1
	iNES2
)

// Archaic version of the iNES format
type iNES0Header struct {
	NESMagic    int32 // NESMagicConstant
	PRG_ROMSize byte  // in 16kB units
	CHR_ROMSize byte  // in 8kB units (0 means the board uses CHR RAM)
	Flags6      byte  // Mapper, mirroring, battery, trainer
}

type iNES interface {
	Config() iNESConfig
}

// version 1.0 of the iNES format
type iNES1Header struct {
	iNES0Header
	Flags7  byte    // Mapper, VS/PlayChoice, NES 2.0
	Flags8  byte    // PRG-RAM size
	Flags9  byte    // TV System
	Padding [6]byte // should be zero filled
}

// version 2.0 of the iNES format
type iNES2Header struct {
	iNES0Header
	Flags7  byte // Mapper high nibble, NES 2.0 signature, PlayChoice, Vs
	Flags8  byte // Mapper highest nibble, mapper variant
	Flags9  byte // Upper bits of ROM size
	Flags10 byte // PRG RAM size (logarithmic; battery and non-battery)
	Flags11 byte // VRAM size (logarithmic; battery and non-battery)
	Flags12 byte // TV system
	Flags13 byte // Vs. PPU variant
	Flags14 byte // Miscellaneous ROMs
	Flags15 byte // Default expansion device
}

type iNESHeader struct {
	Flags [16]byte
}

type iNESConfig struct {
	mapper  byte
	mirror  byte
	battery bool
	trainer bool
	prgSize int
	chrSize int
	ramSize int // in 8KiB chunks
}

func (h *iNESHeader) MagicNumber() int32 {
	return int32(h.Flags[3])<<24 |
		int32(h.Flags[2])<<16 |
		int32(h.Flags[1])<<8 |
		int32(h.Flags[0])
}

func (h *iNESHeader) Version() (iNESFormat, error) {
	if h.MagicNumber() != NESMagicConstant {
		return iNESInvalid, fmt.Errorf(fmt.Sprintf("muggle iNES file, wrong magic number: %v", h.MagicNumber()))
	}

	version := iNESFormat(iNES0)
	if (h.Flags[7] & 0x0C) == 0x8 {
		version = iNES2
	} else if (h.Flags[7] & 0xC) == 0 {
		allZero := true
		for i := 12; (i < 16) && allZero; i++ {
			if h.Flags[i] != 0 {
				allZero = false
			}
		}
		if allZero {
			version = iNES1
		}
	}

	return version, nil
}

func (h *iNESHeader) Config() (iNESConfig, error) {
	iNes, err := NewINES(h)
	if err != nil {
		return iNESConfig{}, err
	}
	return iNes.Config(), nil
}

func NewINES(header *iNESHeader) (iNES, error) {
	version, err := header.Version()
	if err != nil {
		return nil, err
	}

	switch version {
	case iNES0:
		h := new(iNES0Header)
		*h = *(*iNES0Header)(unsafe.Pointer(header))
		return h, nil
	case iNES1:
		h := new(iNES1Header)
		*h = *(*iNES1Header)(unsafe.Pointer(header))
		return h, nil
	case iNES2:
		h := new(iNES2Header)
		*h = *(*iNES2Header)(unsafe.Pointer(header))
		return h, nil
	default:
		// should already be validated
		panic(fmt.Sprintf("iNES type %v not implemented!", version))
	}
}

func (h *iNES0Header) Config() iNESConfig {
	mirror1 := h.Flags6 & 1
	mirror2 := (h.Flags6 >> 3) & 1

	return iNESConfig{
		mapper:  h.Flags6 >> 4,
		mirror:  mirror1 | mirror2<<1,
		battery: ((h.Flags6 >> 1) & 1) == 1,
		trainer: h.Flags6&4 == 4,
		prgSize: int(h.PRG_ROMSize) * 16384,
		chrSize: int(h.CHR_ROMSize) * 8192,
		ramSize: 8192,
	}
}

func (h *iNES1Header) Config() iNESConfig {
	mapper1 := h.Flags6 >> 4
	mapper2 := h.Flags7 >> 4
	mirror1 := h.Flags6 & 1
	mirror2 := (h.Flags6 >> 3) & 1

	if h.Flags8 == 0 {
		// Value 0 infers 1 (8 KB) for compatibility; see PRG RAM circuit)
		h.Flags8 = 1
	}
	ramSize := int(h.Flags8) * 8192

	return iNESConfig{
		mapper:  mapper1 | mapper2<<4,
		mirror:  mirror1 | mirror2<<1,
		battery: ((h.Flags6 >> 1) & 1) == 1,
		trainer: h.Flags6&4 == 4,
		prgSize: int(h.PRG_ROMSize) * 16384,
		chrSize: int(h.CHR_ROMSize) * 8192,
		ramSize: ramSize,
	}
}

func (h *iNES2Header) Config() iNESConfig {
	mapper1 := h.Flags6 >> 4
	mapper2 := h.Flags7 >> 4
	mirror1 := h.Flags6 & 1
	mirror2 := (h.Flags6 >> 3) & 1

	if h.Flags8 == 0 {
		// Value 0 infers 1 (8 KB) for compatibility; see PRG RAM circuit)
		h.Flags8 = 1
	}
	ramSize := int(h.Flags8) * 8192

	return iNESConfig{
		mapper:  mapper1 | mapper2<<4,
		mirror:  mirror1 | mirror2<<1,
		battery: ((h.Flags6 >> 1) & 1) == 1,
		trainer: h.Flags6&4 == 4,
		prgSize: int(h.PRG_ROMSize) * 16384,
		chrSize: int(h.CHR_ROMSize) * 8192,
		ramSize: ramSize,
	}
}
