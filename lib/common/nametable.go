package common

import (
	"fmt"
)

type NameTableMirroring uint8

// NameTable Mirroring
const (
	HorizontalMirroring NameTableMirroring = iota
	VerticalMirroring
	SingleScreenMirroring
	// To support this mode we might either need another module in the mapper or
	// give this one access to the cartridge
	QuadScreenMirroring
	QuadScreenMirroringOnly
)

// busInt
type NameTables struct {
	vRam Ram

	Mirroring NameTableMirroring
}

func (n *NameTables) Serialise(s Serialiser) error {
	return s.Serialise(&n.vRam, n.Mirroring)
}
func (n *NameTables) DeSerialise(s Serialiser) error {
	return s.DeSerialise(&n.vRam, &n.Mirroring)
}

func (n *NameTables) Init(defaultMirror NameTableMirroring) {
	// todo: when to use double (for Quad Mirroring)
	n.vRam.Init(0x800 * 2)
	n.Mirroring = defaultMirror
}

func (n *NameTables) Read8(addr uint16) uint8 {
	addr = n.decode(addr)
	return n.vRam.Read8(addr)
}
func (n *NameTables) Write8(addr uint16, val uint8) {
	addr = n.decode(addr)
	n.vRam.Write8(addr, val)
}

func (n *NameTables) decode(addr uint16) uint16 {
	a := addr
	addr -= 0x2000
	table := addr / 0x400
	addr = addr % 0x400
	switch n.Mirroring {
	case HorizontalMirroring:
		// $2000 equals $2400 and $2800 equals $2C00
		switch table {
		case 0:
			table = 0
		case 1:
			table = 0
		case 2:
			table = 1
		case 3:
			table = 1
		default:
			panic(fmt.Errorf("invalid nametable address %x", a))
		}
	case VerticalMirroring:
		// $2000 equals $2800 and $2400 equals $2C00
		switch table {
		case 0:
			table = 0
		case 1:
			table = 1
		case 2:
			table = 0
		case 3:
			table = 1
		default:
			panic("Invalid nametable address")
		}
	case SingleScreenMirroring:
		// All nametables refer to the same memory at any given time,
		// and the mapper directly manipulates CIRAM address bit 10
		panic("Not implemented")
	case QuadScreenMirroring:
		// CIRAM is disabled, and the cartridge contains additional VRAM used for all nametables
		switch table {
		case 0:
			table = 0
		case 1:
			table = 1
		case 2:
			table = 2
		case 3:
			table = 3
		default:
			panic("Invalid nametable address")
		}
	}
	return (table * 0x400) + addr
}
