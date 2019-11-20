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
	//OAMDMA
)

// http://wiki.nesdev.com/w/index.php/PPU_scrolling
//    First      Second
// /¯¯¯¯¯¯¯¯¯\ /¯¯¯¯¯¯¯\
// 0 0yy NN YY YYY XXXXX
//   ||| || || ||| +++++-- coarse X scroll
//   ||| || ++-+++-------- coarse Y scroll
//   ||| ++--------------- nametable select
//   +++------------------ fine Y scroll
type loopyRegister struct {
	register16
}

func (l *loopyRegister) setNameTables(val uint16) {
	l.val = (l.val & 0xF3FF) | ((val & 0x3) << 10)
}
func (l *loopyRegister) getNameTables() uint16 {
	return (l.val & 0x0C00) >> 10
}
func (l *loopyRegister) flipNameTableH() {
	l.val ^= 0x0400
}
func (l *loopyRegister) flipNameTableV() {
	l.val ^= 0x0800
}

// t: ....... ...HGFED = d: HGFED...
// x:              CBA = d: .....CBA
// w:                  = 1
func (l *loopyRegister) setCoarseX(val uint16) {
	l.val = (l.val & 0xFFE0) | (val & 0x1F)
}
func (l *loopyRegister) getCoarseX() uint16 {
	return l.val & 0x1F
}

// t: CBA..HG FED..... = d: HGFEDCBA
// w:                  = 01
func (l *loopyRegister) setCoarseY(val uint16) {
	l.val = (l.val & 0xFC1F) | ((val & 0x1F) << 5)
}
func (l *loopyRegister) getCoarseY() uint16 {
	return (l.val >> 5) & 0x1F
}

func (l *loopyRegister) setFineY(val uint16) {
	l.val = (l.val & 0x8FFF) | ((val & 0x7) << 12)
}
func (l *loopyRegister) getFineY() uint16 {
	return (l.val >> 12) & 0x7
}

func (l *loopyRegister) setMsb(val uint8) {
	l.val = (l.val & 0x80FF) | ((uint16(val) & 0x3F) << 8)
}
func (l *loopyRegister) setLsb(val uint8) {
	l.val = (l.val & 0xFF00) | uint16(val)
}

func (l *loopyRegister) copy(t loopyRegister) {
	l.val = t.val
}
func (l *loopyRegister) copyHori(t loopyRegister) {
	// v: ....F.. ...EDCBA = t: ....F.. ...EDCBA
	l.val = (l.val & 0xFBE0) | (t.val & 0x041F)
}
func (l *loopyRegister) copyVert(t loopyRegister) {
	// v: IHGF.ED CBA..... = t: IHGF.ED CBA.....
	l.val = (l.val & 0x841F) | (t.val & 0x7BE0)
}

// 1 or 32 <=> y pixel or nametable?
func (l *loopyRegister) inc(val uint16) {
	l.val += val
}

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
func (p *Ppu) getBaseNameTable() uint16 {
	return 0x2000 + uint16(p.regs[PPUCTRL].val&0x3)*0x400
}

func (p *Ppu) getVRAMAddrInc() uint16 {
	if p.regs[PPUCTRL].val&4 == 0 {
		return 1
	} else {
		return 32
	}
}

func (p *Ppu) getSpritePattern() uint16 {
	_, spriteYSize := p.getSpriteSize()
	if spriteYSize == 8 {
		return (uint16(p.regs[PPUCTRL].read()&8) >> 3) * 0x1000
	}
	return 0x1000
}

func (p *Ppu) getBackgroundTable() uint16 {
	return (uint16(p.regs[PPUCTRL].val&16) >> 4) * 0x1000
}

func (p *Ppu) getSpriteSize() (uint8, uint8) {
	return 8, (((p.regs[PPUCTRL].val >> 5) & 0x1) * 8) + 8
}

func (p *Ppu) getMasterSlaveSelect() uint8 {
	return (p.regs[PPUCTRL].val & 64) >> 6
}

func (p *Ppu) getNMIVertical() uint8 {
	return (p.regs[PPUCTRL].val & 128) >> 7
}

func (p *Ppu) writeControl() {
	val := p.regs[PPUCTRL].val

	// t: ....BA.. ........ = d: ......BA
	p.tRAM.setNameTables(uint16(val))
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
	return p.regs[PPUMASK].val & 1
}

func (p *Ppu) showBackgroundLeft() uint8 {
	return (p.regs[PPUMASK].val & 2) >> 1
}

func (p *Ppu) showSpritesLeft() uint8 {
	return (p.regs[PPUMASK].val & 4) >> 2
}

func (p *Ppu) showBackground() bool {
	return ((p.regs[PPUMASK].val & 8) >> 3) == 1
}

func (p *Ppu) showSprites() bool {
	return ((p.regs[PPUMASK].val & 16) >> 4) == 1
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

const (
	statusSpriteOverflow = 1 << 5
	statusSprite0Hit     = 1 << 6
)

func (p *Ppu) setSTATUSbits(val uint8) {
	p.regs[PPUSTATUS].set(val)
}

func (p *Ppu) setLastRegWrite(val uint8) {
	// Least significant bits previously written into a PPU register
	p.regs[PPUSTATUS].val = (p.regs[PPUSTATUS].val & 0xE0) | (val & 0x1F)
}

func (p *Ppu) readPPUStatus() uint8 {
	val := p.regs[PPUSTATUS].val

	p.clear(cpuIntNMI) // clear vblank
	p.wToggle.val = 0
	// Race Condition Warning: Reading PPUSTATUS within two cycles of the start of vertical blank will return 0
	// in bit 7 but clear the latch anyway, causing NMI to not occur that frame. See NMI and PPU_frame_timing for details.
	return val
}

func (p *Ppu) writePPUScroll() {
	val := p.regs[PPUSCROLL].read()
	if p.wToggle.val == 0 {
		// t: ....... ...HGFED = d: HGFED...
		// x:              CBA = d: .....CBA
		// w:                  = 1
		p.tRAM.setCoarseX(uint16(val) >> 3)
		p.xFine.write(val & 0x7)

		p.xScroll = val // for the dummy version
		p.wToggle.val = 1
	} else {
		// t: CBA..HG FED..... = d: HGFEDCBA
		// w:                  = 01
		p.tRAM.setFineY(uint16(val))
		p.tRAM.setCoarseY(uint16(val) >> 3)

		p.wToggle.val = 0
	}
}

func (p *Ppu) writePPUAddr() {
	val := p.regs[PPUADDR].read()
	if p.wToggle.val == 0 {
		// http://wiki.nesdev.com/w/index.php/PPU_scrolling:
		// t: .FEDCBA ........ = d: ..FEDCBA
		// t: X...... ........ = 0
		// w:                  = 1
		p.tRAM.setMsb(val)
		p.wToggle.val = 1
	} else {
		// t: ....... HGFEDCBA = d: HGFEDCBA
		// v                   = t
		// w:                  = 0
		p.tRAM.setLsb(val)
		p.vRAM.copy(p.tRAM)
		p.wToggle.val = 0
	}
}

func (p *Ppu) readPPUData() uint8 {
	val := p.busInt.read8(p.vRAM.val)

	// https://wiki.nesdev.com/w/index.php/PPU_registers#PPUSTATUS
	// When reading while the VRAM address is in the range 0-$3EFF (i.e., before the palettes), the read will return
	// the contents of an internal read buffer. This internal buffer is updated only when reading PPUDATA, and so is
	// preserved across frames. After the CPU reads and gets the contents of the internal buffer, the PPU will
	// immediately update the internal buffer with the byte at the current VRAM address.
	// Thus, after setting the VRAM address, one should first read this register and discard the result.
	if p.vRAM.val%0x4000 < 0x3F00 {
		p.vRAMBuffer, val = val, p.vRAMBuffer
	} else {
		p.vRAMBuffer = p.busInt.read8(p.vRAM.val - 0x1000)
	}
	p.regs[PPUDATA].val = val

	p.vRAM.inc(p.getVRAMAddrInc())

	return val
}
func (p *Ppu) writePPUData() {
	val := p.regs[PPUDATA].val
	p.busInt.write8(p.vRAM.val, val)

	p.vRAM.inc(p.getVRAMAddrInc())
}

func (p *Ppu) readOAMData() uint8 {
	addr := p.regs[OAMADDR].val
	val := p.rOAM.read8(uint16(addr))
	p.regs[OAMDATA].val = val

	p.regs[OAMADDR].val = addr + 1

	return val
}
func (p *Ppu) writeOAMData() {
	addr := p.regs[OAMADDR].val
	p.rOAM.write8(uint16(addr), p.regs[OAMDATA].val)
	p.regs[OAMADDR].val = addr + 1
}

func (p *Ppu) initRegisters() {

	// external CPU mapped registers
	p.regs[PPUCTRL].initx("PPUCTRL", 0, p.writeControl, nil)
	p.regs[PPUMASK].initx("PPUMASK", 0, nil, nil)
	p.regs[PPUSTATUS].initx("PPUSTATUS", 0, nil, p.readPPUStatus)
	p.regs[OAMADDR].initx("OAMADDR", 0, nil, nil)
	p.regs[OAMDATA].initx("OAMDATA", 0, p.writeOAMData, p.readOAMData)
	p.regs[PPUSCROLL].initx("PPUSCROLL", 0, p.writePPUScroll, nil)
	p.regs[PPUADDR].initx("PPUADDR", 0, p.writePPUAddr, nil)
	p.regs[PPUDATA].initx("PPUDATA", 0, p.writePPUData, p.readPPUData)

	// internal registers
	p.vRAM.init("v", 0)
	p.tRAM.init("t", 0)
	p.xFine.init("x", 0)
	p.wToggle.init("w", 0)
}
