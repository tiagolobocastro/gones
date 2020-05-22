package gones

import "fmt"

type NameTableMirroring uint8

// NameTable Mirroring
const (
	HorizontalMirroring = iota
	VerticalMirroring
	SingleScreenMirroring
	// To support this mode we might either need another module in the mapper or
	// give this one access to the cartridge
	QuadScreenMirroring
)

// busInt
type NameTables struct {
	vRam ram

	mirroring NameTableMirroring
}

func (n *NameTables) init(defaultMirror NameTableMirroring) {
	n.vRam.init(0x800)
	n.mirroring = defaultMirror
}

func (n *NameTables) read8(addr uint16) uint8 {
	addr = n.decode(addr)
	return n.vRam.read8(addr)
}
func (n *NameTables) write8(addr uint16, val uint8) {
	addr = n.decode(addr)
	n.vRam.write8(addr, val)
}

func (n *NameTables) decode(addr uint16) uint16 {
	a := addr
	addr -= 0x2000
	table := addr / 0x400
	addr = addr % 0x400
	switch n.mirroring {
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
		panic("Not implemented")
	}
	return (table * 0x400) + addr
}
