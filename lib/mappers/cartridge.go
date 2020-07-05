package mappers

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/tiagolobocastro/gones/lib/common"
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
	Tick()
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
	if c.config.battery {
		c.prgRam.LoadFromFile(c.getRamSaveFile())
	}

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

func (c *Cartridge) Ticks(nTicks int) {
	for i := 0; i < nTicks; i++ {
		c.Mapper.Tick()
	}
}

func (c *Cartridge) Stop() {
	if c.config.battery {
		if err := c.prgRam.SaveToFile(c.getRamSaveFile()); err != nil {
			log.Panicf("Failed to save game: %v", err)
		}
	}
}

func (c *Cartridge) Reset() {
	c.Init(c.cart)
}

func (c *Cartridge) Serialise(s common.Serialiser) error {
	return s.Serialise(c.prgRom, c.prgRam, c.chr, c.ram, &c.Tables, c.Mapper)
}
func (c *Cartridge) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(c.prgRom, c.prgRam, c.chr, c.ram, &c.Tables, c.Mapper)
}

func (c *Cartridge) newCartMapper(mapper byte) Mapper {
	switch mapper {
	case 0:
		return &MapperNROM{cart: c}
	case 1:
		return &MapperMMC1{cart: c}
	case 2, 9:
		return &MapperMMC2{cart: c}
	case 4:
		return &MapperMMC3{cart: c}
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

// must be called after the prgRom is loaded
func (c *Cartridge) getRamSaveFile() *os.File {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panicf("Failed to get user homedir: %v", err)
	}
	_, romName := filepath.Split(c.cart)
	// adding a a hash of the prgRom to help since I tend to use tmp images ("a.nes") for debugging ease
	saveFolder := fmt.Sprintf("%s/.config/gones", homeDir)
	save := fmt.Sprintf("%s/%s_%x", saveFolder, romName, c.prgRom.Hash())
	if _, err := os.Stat(save); os.IsNotExist(err) {
		if err := os.MkdirAll(saveFolder, 0700); err != nil {
			log.Panicf("Failed to create save folder: %v", err)
		}
		f, err := os.Create(save)
		if err != nil {
			log.Panicf("Failed to create save file: %v", err)
		}
		f.Close()
	}
	f, err := os.Open(save)
	if err != nil {
		log.Panicf("Failed to open save file: %v", err)
	}
	return f
}

func (c *Cartridge) GetStateSaveFile() *os.File {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Panicf("Failed to get user homedir: %v", err)
	}
	_, romName := filepath.Split(c.cart)
	// adding a a hash of the prgRom to help since I tend to use tmp images ("a.nes") for debugging ease
	saveFolder := fmt.Sprintf("%s/.config/gones", homeDir)
	save := fmt.Sprintf("%s/%s_%x", saveFolder, romName, c.prgRom.Hash())
	if _, err := os.Stat(save); os.IsNotExist(err) {
		if err := os.MkdirAll(saveFolder, 0700); err != nil {
			log.Panicf("Failed to create save folder: %v", err)
		}
		f, err := os.Create(save)
		if err != nil {
			log.Panicf("Failed to create state save file: %v", err)
		}
		f.Close()
	}
	f, err := os.OpenFile(save, os.O_CREATE|os.O_RDWR, os.ModeExclusive)
	if err != nil {
		log.Panicf("Failed to open state save file: %v", err)
	}
	return f
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
