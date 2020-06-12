package ppu

import (
	"github.com/tiagolobocastro/gones/nes/common"
	"github.com/tiagolobocastro/gones/nes/cpu"
)

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
	common.Register16
}

func (l *loopyRegister) setNameTables(val uint16) {
	l.Val = (l.Val & 0xF3FF) | ((val & 0x3) << 10)
}
func (l *loopyRegister) getNameTables() uint16 {
	return (l.Val & 0x0C00) >> 10
}
func (l *loopyRegister) flipNameTableH() {
	l.Val ^= 0x0400
}
func (l *loopyRegister) flipNameTableV() {
	l.Val ^= 0x0800
}

// t: ....... ...HGFED = d: HGFED...
// X:              CBA = d: .....CBA
// w:                  = 1
func (l *loopyRegister) setCoarseX(val uint16) {
	l.Val = (l.Val & 0xFFE0) | (val & 0x1F)
}
func (l *loopyRegister) getCoarseX() uint16 {
	return l.Val & 0x1F
}

// t: CBA..HG FED..... = d: HGFEDCBA
// w:                  = 01
func (l *loopyRegister) setCoarseY(val uint16) {
	l.Val = (l.Val & 0xFC1F) | ((val & 0x1F) << 5)
}
func (l *loopyRegister) getCoarseY() uint16 {
	return (l.Val >> 5) & 0x1F
}

func (l *loopyRegister) setFineY(val uint16) {
	l.Val = (l.Val & 0x8FFF) | ((val & 0x7) << 12)
}
func (l *loopyRegister) getFineY() uint16 {
	return (l.Val >> 12) & 0x7
}

func (l *loopyRegister) setMsb(val uint8) {
	l.Val = (l.Val & 0x80FF) | ((uint16(val) & 0x3F) << 8)
}
func (l *loopyRegister) setLsb(val uint8) {
	l.Val = (l.Val & 0xFF00) | uint16(val)
}

func (l *loopyRegister) copy(t loopyRegister) {
	l.Val = t.Val
}
func (l *loopyRegister) copyHori(t loopyRegister) {
	// v: ....F.. ...EDCBA = t: ....F.. ...EDCBA
	l.Val = (l.Val & 0xFBE0) | (t.Val & 0x041F)
}
func (l *loopyRegister) copyVert(t loopyRegister) {
	// v: IHGF.ED CBA..... = t: IHGF.ED CBA.....
	l.Val = (l.Val & 0x841F) | (t.Val & 0x7BE0)
}

// 1 or 32 <=> Y pixel or nametable?
func (l *loopyRegister) inc(val uint16) {
	l.Val += val
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
	return 0x2000 + uint16(p.regs[PPUCTRL].Val&0x3)*0x400
}

func (p *Ppu) getVRAMAddrInc() uint16 {
	if p.regs[PPUCTRL].Val&4 == 0 {
		return 1
	} else {
		return 32
	}
}

func (p *Ppu) getSpritePattern() uint16 {
	_, spriteYSize := p.getSpriteSize()
	if spriteYSize == 8 {
		return (uint16(p.regs[PPUCTRL].Read()&8) >> 3) * 0x1000
	}
	return 0x1000
}

func (p *Ppu) getBackgroundTable() uint16 {
	return (uint16(p.regs[PPUCTRL].Val&16) >> 4) * 0x1000
}

func (p *Ppu) getSpriteSize() (uint8, uint8) {
	return 8, (((p.regs[PPUCTRL].Val >> 5) & 0x1) * 8) + 8
}

func (p *Ppu) getMasterSlaveSelect() uint8 {
	return (p.regs[PPUCTRL].Val & 64) >> 6
}

func (p *Ppu) getNMIVertical() uint8 {
	return (p.regs[PPUCTRL].Val & 128) >> 7
}

func (p *Ppu) writeControl() {
	val := p.regs[PPUCTRL].Val

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
	return p.regs[PPUMASK].Val & 1
}

func (p *Ppu) showBackgroundLeft() bool {
	return (p.regs[PPUMASK].Val & 2) != 0
}

func (p *Ppu) showSpritesLeft() bool {
	return (p.regs[PPUMASK].Val & 4) != 0
}

func (p *Ppu) showBackground() bool {
	return ((p.regs[PPUMASK].Val & 8) >> 3) == 1
}

func (p *Ppu) showSprites() bool {
	return ((p.regs[PPUMASK].Val & 16) >> 4) == 1
}

// 0 R G B
func (p *Ppu) showEmphasize() uint8 {
	return (p.regs[PPUMASK].Val & 0xE0) >> 5
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
	p.regs[PPUSTATUS].Set(val)
}

func (p *Ppu) setLastRegWrite(val uint8) {
	// Least significant bits previously written into a PPU register
	p.regs[PPUSTATUS].Val = (p.regs[PPUSTATUS].Val & 0xE0) | (val & 0x1F)
}

func (p *Ppu) readPPUStatus() uint8 {
	val := p.regs[PPUSTATUS].Val

	p.clear(cpu.CpuIntNMI) // clear vblank
	p.wToggle.Val = 0
	// Race Condition Warning: Reading PPUSTATUS within two cycles of the start of vertical blank will return 0
	// in bit 7 but clear the latch anyway, causing NMI to not occur that frame. See NMI and PPU_frame_timing for details.
	return val
}

func (p *Ppu) writePPUScroll() {
	val := p.regs[PPUSCROLL].Read()
	if p.wToggle.Val == 0 {
		// t: ....... ...HGFED = d: HGFED...
		// X:              CBA = d: .....CBA
		// w:                  = 1
		p.tRAM.setCoarseX(uint16(val) >> 3)
		p.xFine.Write(val & 0x7)

		p.xScroll = val // for the dummy version
		p.wToggle.Val = 1
	} else {
		// t: CBA..HG FED..... = d: HGFEDCBA
		// w:                  = 01
		p.tRAM.setFineY(uint16(val))
		p.tRAM.setCoarseY(uint16(val) >> 3)

		p.wToggle.Val = 0
	}
}

func (p *Ppu) writePPUAddr() {
	val := p.regs[PPUADDR].Read()
	if p.wToggle.Val == 0 {
		// http://wiki.nesdev.com/w/index.php/PPU_scrolling:
		// t: .FEDCBA ........ = d: ..FEDCBA
		// t: X...... ........ = 0
		// w:                  = 1
		p.tRAM.setMsb(val)
		p.wToggle.Val = 1
	} else {
		// t: ....... HGFEDCBA = d: HGFEDCBA
		// v                   = t
		// w:                  = 0
		p.tRAM.setLsb(val)
		p.vRAM.copy(p.tRAM)
		p.wToggle.Val = 0
	}
}

func (p *Ppu) readPPUData() uint8 {
	val := p.BusInt.Read8(p.vRAM.Val)

	// https://wiki.nesdev.com/w/index.php/PPU_registers#PPUSTATUS
	// When reading while the VRAM address is in the range 0-$3EFF (i.e., before the palettes), the read will return
	// the contents of an internal read buffer. This internal buffer is updated only when reading PPUDATA, and so is
	// preserved across frames. After the CPU reads and gets the contents of the internal buffer, the PPU will
	// immediately update the internal buffer with the byte at the current VRAM address.
	// Thus, after setting the VRAM address, one should first read this register and discard the result.
	if p.vRAM.Val%0x4000 < 0x3F00 {
		p.vRAMBuffer, val = val, p.vRAMBuffer
	} else {
		p.vRAMBuffer = p.BusInt.Read8(p.vRAM.Val - 0x1000)
	}
	p.regs[PPUDATA].Val = val

	p.vRAM.inc(p.getVRAMAddrInc())

	return val
}
func (p *Ppu) writePPUData() {
	val := p.regs[PPUDATA].Val
	p.BusInt.Write8(p.vRAM.Val, val)

	p.vRAM.inc(p.getVRAMAddrInc())
}

func (p *Ppu) readOAMData() uint8 {
	addr := p.regs[OAMADDR].Val
	val := p.rOAM.Read8(uint16(addr))
	p.regs[OAMDATA].Val = val

	p.regs[OAMADDR].Val = addr + 1

	return val
}
func (p *Ppu) writeOAMData() {
	addr := p.regs[OAMADDR].Val
	p.rOAM.Write8(uint16(addr), p.regs[OAMDATA].Val)
	p.regs[OAMADDR].Val = addr + 1
}

func (p *Ppu) initRegisters() {

	// external CPU mapped registers
	p.regs[PPUCTRL].Initx("PPUCTRL", 0, p.writeControl, nil)
	p.regs[PPUMASK].Initx("PPUMASK", 0, nil, nil)
	p.regs[PPUSTATUS].Initx("PPUSTATUS", 0, nil, p.readPPUStatus)
	p.regs[OAMADDR].Initx("OAMADDR", 0, nil, nil)
	p.regs[OAMDATA].Initx("OAMDATA", 0, p.writeOAMData, p.readOAMData)
	p.regs[PPUSCROLL].Initx("PPUSCROLL", 0, p.writePPUScroll, nil)
	p.regs[PPUADDR].Initx("PPUADDR", 0, p.writePPUAddr, nil)
	p.regs[PPUDATA].Initx("PPUDATA", 0, p.writePPUData, p.readPPUData)

	// internal registers
	p.vRAM.Init("v", 0)
	p.tRAM.Init("t", 0)
	p.xFine.Init("X", 0)
	p.wToggle.Init("w", 0)
}
