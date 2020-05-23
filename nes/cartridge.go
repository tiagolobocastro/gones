package gones

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var cartEndianness = binary.LittleEndian

func (c *Cartridge) defaultInit() error {
	c.prgRom.init(16384*4, true)
	c.chr.init(16384, true)
	c.ram.init(16384)

	c.mapper = c.newCartMapper(mapperNROM)

	return nil
}

func (c *Cartridge) init(cartPath string) error {
	c.cart = cartPath

	c.prgRom = new(rom)
	c.prgRam = new(ram)
	c.chr = new(rom)
	c.ram = new(ram)

	if c.cart == "" {
		// current go tests do not use a cartridge but rather just
		// soft load code on demand
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

	c.config, err = header.Config()
	if err != nil {
		return err
	}

	if c.config.trainer {
		trainer := make([]byte, 512)
		if _, err = io.ReadFull(file, trainer); err != nil {
			return err
		}
	}

	c.prgRom.init(c.config.prgRomSize, false)
	if _, err = io.ReadFull(file, c.prgRom.rom); err != nil {
		return err
	}
	c.prgRam.init(c.config.prgRamSize)

	c.chr.init(c.config.chrRomSize, false)
	if _, err = io.ReadFull(file, c.chr.rom); err != nil {
		return err
	}
	if c.config.chrRomSize == 0 {
		c.chr.init(0x4000, true)
	}

	c.mapper = c.newCartMapper(c.config.mapper)
	c.mapper.Init()
	c.tables.init(NameTableMirroring(c.config.mirror))
	return nil
}

func (c *Cartridge) newCartMapper(mapper byte) Mapper {
	switch mapper {
	case mapperNROM:
		return &MapperNROM{cart: c}
	case mapperMMC1:
		return &MapperMMC1{cart: c}
	case mapperUnROM:
		return &MapperNROM{cart: c}
	default:
		panic(fmt.Sprintf("mapper %v not supported!", mapper))
	}
}

func (c *Cartridge) SetMirroring(mirroring NameTableMirroring) {
	c.tables.mirroring = mirroring
}

// BusInt
type Cartridge struct {
	config  iNESConfig
	version iNESFormat
	cart    string

	prgRom *rom
	prgRam *ram
	chr    *rom
	ram    *ram
	tables NameTables

	mapper Mapper
}

// loads hex dumps from: https://skilldrick.github.io/easy6502/, eg:
// `0600: a9 01 85 02 a9 cc 8d 00 01 a9 01 a a1 00 00 00
//  0610: a9 05 a 8e 00 02 a9 05 8d 01 02 a9 08 8d 02 02`

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
			n.cart.prgRom.write16(0xFFFC, uint16(addr))
		}

		for i, b := range bt {
			n.cpu.write8(uint16(addr+i), uint8(b))
		}
	}
}
