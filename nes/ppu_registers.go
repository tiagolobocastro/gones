package gones

const (
	PPUCTRL = iota
	PPUMASK
	PPUSTATUS
	OAMADDR
	OAMDATA
	PPUSCROLL
	PPUADDR
	PPUDATA
	OAMDMA
)

/* PPUCTRL
7  bit  0
---- ----
VPHB SINN
|||| ||||
|||| ||++- Base nametable address
|||| ||    (0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00)
|||| |+--- VRAM address increment per CPU read/write of PPUDATA
|||| |     (0: add 1, going across; 1: add 32, going down)
|||| +---- Sprite pattern table address for 8x8 sprites
||||       (0: $0000; 1: $1000; ignored in 8x16 mode)
|||+------ Background pattern table address (0: $0000; 1: $1000)
||+------- Sprite size (0: 8x8 pixels; 1: 8x16 pixels)
|+-------- PPU master/slave select
|          (0: read backdrop from EXT pins; 1: output color on EXT pins)
+--------- Generate an NMI at the start of the
           vertical blanking interval (0: off; 1: on)
*/
func (p *Ppu) getBaseNametable() uint16 {
	return 0x2000 + uint16(p.regs[PPUCTRL].val&0x3)*0x400
}

func (p *Ppu) getVRAMAddrInc() uint8 {
	if p.regs[PPUCTRL].val&4 == 0 {
		return 1
	} else {
		return 32
	}
	// need to implement PPUDATA
}

func (p *Ppu) getSpritePattern() uint16 {
	return (uint16(p.regs[PPUCTRL].val&8) >> 3) * 0x1000
}

func (p *Ppu) getBackgroundTable() uint16 {
	return (uint16(p.regs[PPUCTRL].val&16) >> 4) * 0x1000
}

func (p *Ppu) getSpriteSize() (uint8, uint8) {
	return 8, ((p.regs[PPUCTRL].val & 5) >> 8) * 8
}

func (p *Ppu) getMasterSlaveSelect() uint8 {
	return (p.regs[PPUCTRL].val & 64) >> 6
}

func (p *Ppu) getNMIVertical() uint8 {
	return (p.regs[PPUCTRL].val & 128) >> 7
}

/* PPU Mask
7  bit  0
---- ----
BGRs bMmG
|||| ||||
|||| |||+- Greyscale (0: normal color, 1: produce a greyscale display)
|||| ||+-- 1: Show background in leftmost 8 pixels of screen, 0: Hide
|||| |+--- 1: Show sprites in leftmost 8 pixels of screen, 0: Hide
|||| +---- 1: Show background
|||+------ 1: Show sprites
||+------- Emphasize red
|+-------- Emphasize green
+--------- Emphasize blue
*/

func (p *Ppu) getGreyScale() uint8 {
	return (p.regs[PPUMASK].val & 1)
}

func (p *Ppu) showBackgroundLeft() uint8 {
	return (p.regs[PPUMASK].val & 2) >> 1
}

func (p *Ppu) showSpritesLeft() uint8 {
	return (p.regs[PPUMASK].val & 4) >> 2
}

func (p *Ppu) showBackground() uint8 {
	return (p.regs[PPUMASK].val & 8) >> 3
}

func (p *Ppu) showSprites() uint8 {
	return (p.regs[PPUMASK].val & 16) >> 4
}

// 0 R G B
func (p *Ppu) showEmphasize() uint8 {
	return (p.regs[PPUMASK].val & 0xE0) >> 5
}

/* PPUSTATUS
7  bit  0
---- ----
VSO. ....
|||| ||||
|||+-++++- Least significant bits previously written into a PPU register
|||        (due to register not being updated for this address)
||+------- Sprite overflow. The intent was for this flag to be set
||         whenever more than eight sprites appear on a scanline, but a
||         hardware bug causes the actual behavior to be more complicated
||         and generate false positives as well as false negatives; see
||         PPU sprite evaluation. This flag is set during sprite
||         evaluation and cleared at dot 1 (the second dot) of the
||         pre-render line.
|+-------- Sprite 0 Hit.  Set when a nonzero pixel of sprite 0 overlaps
|          a nonzero background pixel; cleared at dot 1 of the pre-render
|          line.  Used for raster timing.
+--------- Vertical blank has started (0: not in vblank; 1: in vblank).
           Set at dot 1 of line 241 (the line *after* the post-render
           line); cleared after reading $2002 and at dot 1 of the
           pre-render line.
*/

func (p *Ppu) getSTATUS() uint8 {
	val := p.regs[PPUSTATUS].val
	p.clear(cpuIntNMI) // clear vblank
	p.wToggle.val = 0
	// Race Condition Warning: Reading PPUSTATUS within two cycles of the start of vertical blank will return 0
	// in bit 7 but clear the latch anyway, causing NMI to not occur that frame. See NMI and PPU_frame_timing for details.
	return val
}

// yeah, these should really be func(ppu) assigned to the cpu mapped registers
func (p *Ppu) writePPUAddr(val uint8) {
	if p.wToggle.val == 0 {
		// http://wiki.nesdev.com/w/index.php/PPU_scrolling:
		// t: .FEDCBA ........ = d: ..FEDCBA
		// t: X...... ........ = 0
		// w:                  = 1
		p.tRAM.val = (p.tRAM.val & 0x80FF) | ((uint16(val) & 0x3F) << 8)
		p.wToggle.val = 1
	} else {
		// t: ....... HGFEDCBA = d: HGFEDCBA
		// v                   = t
		// w:                  = 0
		p.tRAM.val = (p.tRAM.val & 0xFF00) | uint16(val)
		p.vRAM.val = p.tRAM.val
		p.wToggle.val = 0
	}
}

func (p *Ppu) writePPUData(val uint8) {
	p.busInt.write8(p.vRAM.val, val)
}

func (p *Ppu) writeOAMData(val uint8) {
	p.regs[OAMDATA].val = val
	p.regs[OAMADDR].val += 1
}
