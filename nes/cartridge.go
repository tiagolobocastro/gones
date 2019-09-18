package gones

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const NESMagicConstant = 0x1A53454E

var cartEndianness = binary.LittleEndian

type iNESHeader struct {
	NESMagic    int32   // "NES" + EOF
	PRG_ROMSize byte    // in 16kB units
	CHR_ROMSize byte    // in 8kB units (0 means the board uses CHR RAM)
	Flags6      byte    // Mapper, mirroring, battery, trainer
	Flags7      byte    // Mapper, VS/PlayChoice, NES 2.0
	Flags8      byte    // PRG-RAM size
	Flags9      byte    // TV System
	Flags10     byte    // TV System, PRG-RAM presence
	Padding     [5]byte // should be zero filled
}

func (c *Cartridge) defaultInit() error {
	c.prg.init(16384 * 4)
	c.chr.init(16384)
	c.ram.init(16384)

	c.mapper = newMapper(c, mapperNROM)

	return nil
}

func (c *Cartridge) init(cartPath string) error {

	c.prg = new(rom)
	c.chr = new(rom)
	c.ram = new(ram)

	if cartPath == "" {
		return c.defaultInit()
	}

	file, err := os.Open(cartPath)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Printf("error closing file:%v\nbut ignoring it since it's we didn't write anything...", err)
		}
	}()

	header := iNESHeader{}
	if err := binary.Read(file, cartEndianness, &header); err != nil {
		return err
	}

	if header.NESMagic != NESMagicConstant {
		return errors.New("that's a muggle iNES")
	}

	mapper1 := header.Flags6 >> 4
	mapper2 := header.Flags7 >> 4
	mapper := mapper1 | mapper2<<4

	// mirroring type
	mirror1 := header.Flags6 & 1
	mirror2 := (header.Flags6 >> 3) & 1
	c.mirror = mirror1 | mirror2<<1

	// NV Ram (battery backed or EEPROM)
	c.battery = (header.Flags6 >> 1) & 1

	// read trainer if present (unused)
	if header.Flags6&4 == 4 {
		trainer := make([]byte, 512)
		if _, err = io.ReadFull(file, trainer); err != nil {
			return err
		}
	}

	c.prg.init(uint32(header.PRG_ROMSize) * 16384)
	if _, err = io.ReadFull(file, c.prg.rom); err != nil {
		return err
	}

	c.chr.init(uint32(header.CHR_ROMSize) * 8192)
	if _, err = io.ReadFull(file, c.chr.rom); err != nil {
		return err
	}

	// provide chr-rom/ram if not in file
	if header.CHR_ROMSize == 0 {
		c.chr.init(8192)
	}

	c.ram.init(uint32(header.Flags8))

	c.mapper = newMapper(c, mapper)

	return nil
}

// BusInt
type Cartridge struct {
	prg     *rom
	chr     *rom
	ram     *ram
	mirror  byte
	battery byte

	mapper *Mapper
}

func (c *Cartridge) getMappingTable() []busMemMapEntry {
	return c.mapper.devEntries
}
