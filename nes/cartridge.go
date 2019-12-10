package gones

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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
	c.prg.init(16384*4, true)
	c.chr.init(16384, true)
	c.ram.init(16384)

	c.mapper = c.newCartMapper(mapperNROM)

	return nil
}

func (c *Cartridge) init(cartPath string) error {

	c.cart = cartPath

	c.prg = new(rom)
	c.chr = new(rom)
	c.ram = new(ram)

	if c.cart == "" {
		return c.defaultInit()
	}

	file, err := os.Open(c.cart)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Printf("error closing file:%v\nbut ignoring it since it's we didn't write anything...", err)
		}
	}()

	header := iNESHeader{}
	if err := binary.Read(file, cartEndianness, &header); err != nil {
		return err
	}

	if header.NESMagic != NESMagicConstant {
		return errors.New("rom path points to a muggle iNES file (failed to find the magic number)")
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

	c.prg.init(int(header.PRG_ROMSize)*16384, false)
	if _, err = io.ReadFull(file, c.prg.rom); err != nil {
		return err
	}

	c.chr.init(int(header.CHR_ROMSize)*8192, false)
	if _, err = io.ReadFull(file, c.chr.rom); err != nil {
		return err
	}

	// provide chr-rom/ram if not in file
	if header.CHR_ROMSize == 0 {
		c.chr.init(8192, false)
	}

	c.ram.init(int(header.Flags8))

	c.mapper = c.newCartMapper(mapper)
	return nil
}

func (c *Cartridge) newCartMapper(mapper byte) Mapper {
	switch mapper {
	case mapperNROM:
		return &MapperNROM{cart: c}
	case mapperUnROM:
		return &MapperNROM{cart: c}
	default:
		panic(fmt.Sprintf("mapper %v not supported!", mapper))
	}
}

// BusInt
type Cartridge struct {
	prg     *rom
	chr     *rom
	ram     *ram
	mirror  byte
	battery byte
	prgSize byte
	chrSize byte

	mapper Mapper

	cart string
}

// loads hex dumps from: https://skilldrick.github.io/easy6502/, eg:
// `0600: a9 01 85 02 a9 cc 8d 00 01 a9 01 aa a1 00 00 00
//  0610: a9 05 aa 8e 00 02 a9 05 8d 01 02 a9 08 8d 02 02`

func (n *nes) loadEasyCode(code string) {

	for i, line := range strings.Split(strings.TrimSuffix(code, "\n"), "\n") {
		addr := 0
		var bt [16]int
		ns, err := fmt.Sscanf(line, "%x: %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x ",
			&addr, &bt[0], &bt[1], &bt[2], &bt[3], &bt[4], &bt[5], &bt[6], &bt[7],
			&bt[8], &bt[9], &bt[10], &bt[11], &bt[12], &bt[13], &bt[14], &bt[15])
		if err != nil && err != io.EOF {
			log.Printf("Error when scanning easyCode line, ns: %x, error: %v\n", ns, err)
		}

		if i == 0 {
			// assumes first line is where the program starts
			n.cart.prg.write16(0xFFFC, uint16(addr))
		}

		for i, b := range bt {
			n.cpu.write8(uint16(addr+i), uint8(b))
		}
	}
}
