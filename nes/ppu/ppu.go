package ppu

import "github.com/tiagolobocastro/gones/nes/cpu"

func (p *Ppu) updateShifter() {
	// palette and pixel index
	// a a i i
	p.rowShifter <<= 4
}

// 1 row of aaii*8
func (p *Ppu) buildBgPixelRow() {
	attr := (p.attributeEntry & 0x3) << 2
	for i := uint(0); i < 8; i++ {
		pixel := uint64(attr | (p.highOrderByte>>6)&2 | (p.lowOrderByte>>7)&1)
		p.rowShifter |= pixel << ((7 - i) * 4)
		p.highOrderByte <<= 1
		p.lowOrderByte <<= 1
	}
}

func (p *Ppu) getBgPixel() uint8 {
	return uint8(p.rowShifter >> (32 + ((7 - p.xFine.Val) * 4)))
}

func (p *Ppu) exec() {

	// setup values required for the draw decision
	x := uint8(p.cycle) - 1
	y := uint8(p.scanLine)
	p.bgIndex = 0
	p.bgPalette = 0
	p.fgIndex = 0
	p.fgPalette = 0
	p.fgPriority = false

	// http://wiki.nesdev.com/w/images/d/d1/Ntsc_timing.png
	visibleFrame := p.scanLine >= 0 && p.scanLine < 240
	preRenderLn := p.scanLine == -1
	vBlankLn := p.scanLine == 241
	renderFrame := visibleFrame || preRenderLn
	copyVertCycle := p.cycle >= 280 && p.cycle <= 304
	copyHoriCycle := p.cycle == 257
	incVert := p.cycle == 256

	// cycle 0 is skipped for BG+odd => background and odd sprite frames?
	// cycle 337-340 are unused
	visibleCycle := p.cycle >= 1 && p.cycle <= 256
	bgTileFetch := visibleCycle || (p.cycle >= 321 && p.cycle <= 336)

	if p.showBackground() {
		if renderFrame && bgTileFetch && p.showBackground() {

			if visibleFrame && visibleCycle {
				bgPix := p.getBgPixel()
				p.bgIndex = bgPix & 0x3
				p.bgPalette = (bgPix >> 2) & 0x3
			}

			p.updateShifter()
			switch p.cycle % 8 {
			case 1:
				p.nametableEntry = p.BusInt.Read8(0x2000 | (p.vRAM.Val & 0x0FFF))
			case 3:
				//  NN 1111 YYY XXX
				//  || |||| ||| +++-- high 3 bits of coarse X (X/4)
				//  || |||| +++------ high 3 bits of coarse Y (Y/4)
				//  || ++++---------- attribute offset (960 bytes)
				//  ++--------------- nametable select
				vv := 0x2000 | 0x03C0 | p.vRAM.getNameTables()<<10 | ((p.vRAM.getCoarseY() >> 2) << 3) | (p.vRAM.getCoarseX() >> 2)

				p.attributeEntry = p.BusInt.Read8(vv)

				// BR BL TR TL
				// shift to find the right half nibble
				if (p.vRAM.getCoarseY() & 0x02) != 0 {
					p.attributeEntry >>= 4
				}
				if (p.vRAM.getCoarseX() & 0x02) != 0 {
					p.attributeEntry >>= 2
				}
			case 5:
				p.lowOrderByte = p.BusInt.Read8(p.getBackgroundTable() | uint16(p.nametableEntry)<<4 | p.vRAM.getFineY())
			case 7:
				p.highOrderByte = p.BusInt.Read8(p.getBackgroundTable() | uint16(p.nametableEntry)<<4 | p.vRAM.getFineY() | 8)
			case 0:
				p.buildBgPixelRow()

				// Increment Horizontal(v)
				if p.vRAM.getCoarseX() == 31 {
					p.vRAM.setCoarseX(0)
					p.vRAM.flipNameTableH()
				} else {
					p.vRAM.setCoarseX(p.vRAM.getCoarseX() + 1)
				}
			}
		}

		if renderFrame {
			if incVert {
				// Increment Vertical(v)
				fineY := p.vRAM.getFineY()
				if fineY < 7 {
					p.vRAM.setFineY(p.vRAM.getFineY() + 1)
				} else {
					p.vRAM.setFineY(0)
					y := p.vRAM.getCoarseY()
					if y == 29 {
						y = 0
						p.vRAM.flipNameTableV()
					} else if y == 31 {
						y = 0
					} else {
						y += 1
					}
					p.vRAM.setCoarseY(y)
				}
			}

			if copyHoriCycle {
				// Horizontal(v) = Horizontal(t)
				p.vRAM.copyHori(p.tRAM)
			}
		}

		if preRenderLn && copyVertCycle {
			// Vertical(v) = Vertical(t)
			p.vRAM.copyVert(p.tRAM)
		}
	}

	if visibleFrame && p.showSprites() {
		switch p.cycle {
		// the ppu "works" these every cycle and it might more efficient for us to do the same
		// but now for simplicity let's bundle each task
		case 1:
			p.clearSecOAM()
		case 257:
			p.evalSprites()
		case 321:
			p.loadSprites()
		}

		if visibleCycle {
			for i := range p.pOAM {
				if p.pOAM[i].id == 64 {
					continue
				}

				s := &p.pOAM[i]
				xi := uint(x) - uint(s.xPos)
				if xi < 8 {

					bit := 8 - xi - 1

					b0 := (s.lsbIndex >> bit) & 1
					b1 := (s.msbIndex >> bit) & 1
					p.fgIndex = b0 | (b1 << 1)
					p.fgPriority = (s.attributes>>5)&1 == 0
					p.fgPalette = s.attributes & 0x3

					// non transparent pixel found so "accept" this sprite
					if p.fgIndex != 0 {

						if s.id == 0 && p.bgIndex > 0 && x != 255 {
							p.regs[PPUSTATUS].Set(statusSprite0Hit)
						}

						break
					}
				}
			}
		}
	}

	if visibleFrame && visibleCycle {

		// what gets drawn based on transparency (index==0) and priority
		if p.bgIndex == 0 && p.fgIndex == 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00)])
		} else if p.bgIndex > 0 && p.fgIndex == 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16(p.bgPalette*4+p.bgIndex))])
		} else if p.bgIndex == 0 && p.fgIndex > 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16((p.fgPalette+4)*4+p.fgIndex))])
		} else if p.bgIndex > 0 && p.fgIndex > 0 {
			if p.fgPriority {
				p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16((p.fgPalette+4)*4+p.fgIndex))])
			} else {
				p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16(p.bgPalette*4+p.bgIndex))])
			}
		}
	}

	p.cycle += 1
	if p.cycle > 340 {

		p.scanLine += 1
		p.cycle = 0

		if p.scanLine > 260 {
			p.scanLine = -1
		}
	} else if p.cycle == 1 {
		if vBlankLn {
			p.raise(cpu.CpuIntNMI)
		} else if preRenderLn {
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpu.CpuIntNMI)
			p.regs[PPUSTATUS].Clr(statusSpriteOverflow | statusSprite0Hit)
		}
	}
}