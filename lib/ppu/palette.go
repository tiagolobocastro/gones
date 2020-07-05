package ppu

import (
	"encoding/binary"
	"image/color"
	"log"
	"os"

	"github.com/tiagolobocastro/gones/lib/common"
	"github.com/tiagolobocastro/gones/lib/mappers"
)

type ppuPalette struct {
	// BusInt

	nesPalette [64]color.RGBA

	// 4 for the background and 4 for the sprites
	indexes [32]uint8
}

func (p *ppuPalette) init() {

	// http://www.thealmightyguru.com/Games/Hacking/Wiki/index.php/NES_Palette
	defaultPalette := []uint32{
		0x7C7C7C, 0x0000FC, 0x0000BC, 0x4428BC, 0x940084, 0xA80020, 0xA81000, 0x881400,
		0x503000, 0x007800, 0x006800, 0x005800, 0x004058, 0x000000, 0x000000, 0x000000,
		0xBCBCBC, 0x0078F8, 0x0058F8, 0x6844FC, 0xD800CC, 0xE40058, 0xF83800, 0xE45C10,
		0xAC7C00, 0x00B800, 0x00A800, 0x00A844, 0x008888, 0x000000, 0x000000, 0x000000,
		0xF8F8F8, 0x3CBCFC, 0x6888FC, 0x9878F8, 0xF878F8, 0xF85898, 0xF87858, 0xFCA044,
		0xF8B800, 0xB8F818, 0x58D854, 0x58F898, 0x00E8D8, 0x787878, 0x000000, 0x000000,
		0xFCFCFC, 0xA4E4FC, 0xB8B8F8, 0xD8B8F8, 0xF8B8F8, 0xF8A4C0, 0xF0D0B0, 0xFCE0A8,
		0xF8D878, 0xD8F878, 0xB8F8B8, 0xB8F8D8, 0x00FCFC, 0xF8D8F8, 0x000000, 0x000000,
	}
	_ = defaultPalette

	// converted from FCEUX.pal file
	fceuxPalette := []uint32{
		0x747474, 0x24188c, 0x0000a8, 0x44009c, 0x8c0074, 0xa80010, 0xa40000, 0x7c0800,
		0x402c00, 0x004400, 0x005000, 0x003c14, 0x183c5c, 0x000000, 0x000000, 0x000000,
		0xbcbcbc, 0x0070ec, 0x2038ec, 0x8000f0, 0xbc00bc, 0xe40058, 0xd82800, 0xc84c0c,
		0x887000, 0x009400, 0x00a800, 0x009038, 0x008088, 0x000000, 0x000000, 0x000000,
		0xfcfcfc, 0x3cbcfc, 0x5c94fc, 0xcc88fc, 0xf478fc, 0xfc74b4, 0xfc7460, 0xfc9838,
		0xf0bc3c, 0x80d010, 0x4cdc48, 0x58f898, 0x00e8d8, 0x787878, 0x000000, 0x000000,
		0xfcfcfc, 0xa8e4fc, 0xc4d4fc, 0xd4c8fc, 0xfcc4fc, 0xfcc4d8, 0xfcbcb0, 0xfcd8a8,
		0xfce4a0, 0xe0fca0, 0xa8f0bc, 0xb0fccc, 0x9cfcf0, 0xc4c4c4, 0x000000, 0x000000,
	}
	_ = fceuxPalette

	for i, c := range fceuxPalette {
		r := byte(c >> 16)
		g := byte(c >> 8)
		b := byte(c)
		p.nesPalette[i] = color.RGBA{r, g, b, 0xFF}
	}
}

func (p *ppuPalette) setPalette(source string) error {

	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Printf("error closing file:%v\nbut ignoring it since it's we didn't write anything...", err)
		}
	}()

	loadPalette := [64][3]uint8{}
	if err := binary.Read(file, mappers.CartEndianness, &loadPalette); err != nil {
		return err
	}

	for i, c := range loadPalette {
		r := c[0]
		g := c[1]
		b := c[2]
		a := uint8(0xFF)
		p.nesPalette[i] = color.RGBA{r, g, b, a}
	}

	return nil
}

// https://wiki.nesdev.com/w/index.php/PPU_palettes
// Addresses $3F10/$3F14/$3F18/$3F1C are mirrors of $3F00/$3F04/$3F08/$3F0C
func (p *ppuPalette) Read8(addr uint16) uint8 {
	if addr >= 16 && addr%4 == 0 {
		addr -= 16
	}
	return p.indexes[addr]
}
func (p *ppuPalette) Write8(addr uint16, val uint8) {
	// looks like some games writes val>0x3F
	// not sure why but let's just cap it for now
	val = val & 0x3F
	addr = addr & 0x1F

	if addr >= 16 && addr%4 == 0 {
		addr -= 16
	}
	p.indexes[addr] = val
}

// little endian
func (p *ppuPalette) Read16(uint16) uint16 {
	panic("oops")
}
func (p *ppuPalette) Write16(uint16, uint16) {
	panic("oops")
}

func (p *ppuPalette) Serialise(s common.Serialiser) error {
	return s.Serialise(p.indexes)
}
func (p *ppuPalette) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&p.indexes)
}
