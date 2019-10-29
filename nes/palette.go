package gones

import "image/color"

type ppuPalette struct {
	// busInt

	nesPalette [64]color.RGBA

	// 4 for the background and 4 for the sprites
	colours [8]color.RGBA
}

func (p *ppuPalette) init() {

	// http://www.thealmightyguru.com/Games/Hacking/Wiki/index.php/NES_Palette
	colors := []uint32{
		0x7C7C7C, 0x0000FC, 0x0000BC, 0x4428BC, 0x940084, 0xA80020, 0xA81000, 0x881400,
		0x503000, 0x007800, 0x006800, 0x005800, 0x004058, 0x000000, 0x000000, 0x000000,
		0xBCBCBC, 0x0078F8, 0x0058F8, 0x6844FC, 0xD800CC, 0xE40058, 0xF83800, 0xE45C10,
		0xAC7C00, 0x00B800, 0x00A800, 0x00A844, 0x008888, 0x000000, 0x000000, 0x000000,
		0xF8F8F8, 0x3CBCFC, 0x6888FC, 0x9878F8, 0xF878F8, 0xF85898, 0xF87858, 0xFCA044,
		0xF8B800, 0xB8F818, 0x58D854, 0x58F898, 0x00E8D8, 0x787878, 0x000000, 0x000000,
		0xFCFCFC, 0xA4E4FC, 0xB8B8F8, 0xD8B8F8, 0xF8B8F8, 0xF8A4C0, 0xF0D0B0, 0xFCE0A8,
		0xF8D878, 0xD8F878, 0xB8F8B8, 0xB8F8D8, 0x00FCFC, 0xF8D8F8, 0x000000, 0x000000,
	}

	for i, c := range colors {
		r := byte(c >> 16)
		g := byte(c >> 8)
		b := byte(c)
		p.nesPalette[i] = color.RGBA{r, g, b, 0xFF}
	}
}

// need to map these properly to the nes colours!

func (p *ppuPalette) read8(addr uint16) uint8 {
	colourIndex := addr / 4
	switch addr % 4 {
	case 0:
		return p.colours[colourIndex].R
	case 1:
		return p.colours[colourIndex].G
	case 2:
		return p.colours[colourIndex].B
	case 3:
		return p.colours[colourIndex].A
	}
	return 0
}
func (p *ppuPalette) write8(addr uint16, val uint8) {
	colourIndex := addr / 4
	switch addr % 4 {
	case 0:
		p.colours[colourIndex].R = val
	case 1:
		p.colours[colourIndex].G = val
	case 2:
		p.colours[colourIndex].B = val
	case 3:
		p.colours[colourIndex].A = val
	}
}

// little endian
func (p *ppuPalette) read16(addr uint16) uint16 {
	return 0
}
func (p *ppuPalette) write16(addr uint16, val uint16) {

}
