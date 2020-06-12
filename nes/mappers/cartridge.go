package mappers

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/tiagolobocastro/gones/nes/common"
)

const (
	mapperNROM = iota
	mapperMMC1
	mapperUnROM
	mapperMMC2
)

type Mapper interface {
	common.BusInt
	Init()
}

var CartEndianness = binary.LittleEndian

func (c *Cartridge) defaultInit() error {
	c.prgRom.Init(16384*4, true)
	c.chr.Init(16384, true)
	c.ram.Init(16384)

	c.Mapper = c.newCartMapper(mapperNROM)

	return nil
}

func (c *Cartridge) Init(cartPath string) error {
	c.cart = cartPath

	c.prgRom = new(common.Rom)
	c.prgRam = new(common.Ram)
	c.chr = new(common.Rom)
	c.ram = new(common.Ram)

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
	if err := binary.Read(file, CartEndianness, &header); err != nil {
		return err
	}

	c.config, err = header.Config()
	if err != nil {
		return err
	}

	if c.config.console != consoleNES {
		log.Panicf("Unsupported console type %v", c.config.console)
	}

	if c.config.trainer {
		trainer := make([]byte, 512)
		if _, err = io.ReadFull(file, trainer); err != nil {
			return err
		}
	}

	c.prgRom.Init(c.config.prgRomSize, false)

	if _, err = c.prgRom.LoadFromFile(file); err != nil {
		return err
	}

	c.prgRam.Init(c.config.prgRamSize)

	c.chr.Init(c.config.chrRomSize, false)
	if _, err = c.chr.LoadFromFile(file); err != nil {
		return err
	}
	if c.config.chrRomSize == 0 {
		c.chr.Init(0x4000, true)
	}

	c.Mapper = c.newCartMapper(c.config.mapper)
	c.Mapper.Init()
	c.Tables.Init(common.NameTableMirroring(c.config.mirror))
	return nil
}

func (c *Cartridge) newCartMapper(mapper byte) Mapper {
	switch mapper {
	case 0:
		return &MapperNROM{cart: c}
	case 1:
		return &MapperMMC1{cart: c}
	case 2, 9:
		return &MapperMMC2{cart: c}
	default:
		panic(fmt.Sprintf("mapper %v not supported!", mapper))
	}
}

func (c *Cartridge) SetMirroring(mirroring common.NameTableMirroring) {
	c.Tables.Mirroring = mirroring
}

func (c *Cartridge) WriteRom16(addr uint16, val uint16) {
	c.prgRom.Write16(addr, val)
}

// BusInt
type Cartridge struct {
	config  iNESConfig
	version iNESFormat
	cart    string

	prgRom *common.Rom
	prgRam *common.Ram
	chr    *common.Rom
	ram    *common.Ram
	Tables common.NameTables

	Mapper Mapper
}
