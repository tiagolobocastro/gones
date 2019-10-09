package gones

import (
	"fmt"
)

type Instruction struct {
	opLength     uint8
	opCycles     uint8
	opPageCycles uint8
	addrMode     uint8

	opCode uint8
	opName string

	// evaluator function
	eval func()
	// because can't compare fun() with cpu.unhandled
	implemented bool
}

type Context struct {
	ins *Instruction
	opr uint32
}

const (
	cpuIntNMI = 1
	cpuIntIRQ = 2
)

type interrupt struct {
	flags uint8
}

// interrupt
func (c *Cpu) raise(flag uint8) {
	c.interrupts |= flag
}

func (c *Cpu) clear(flag uint8) {
	c.interrupts &= flag ^ 0xFF
}

type Cpu struct {
	busExtInt
	interrupt

	ins [256]Instruction

	curr Context

	rg  registers
	apu apu

	clk int64

	verbose      bool
	disableBreak bool

	inInt      bool
	interrupts uint8
}

func (c *Cpu) init(busInt busExtInt, verbose bool) {
	c.verbose = verbose
	c.disableBreak = true

	c.rg.init()
	c.setupIns()

	c.busExtInt = busInt
}

func (c *Cpu) reset() {
	c.rg.init()
	c.inInt = false
	c.rg.spc.pc.write(c.read16(0xFFFC))
}

func (c *Cpu) tick() {

	c.clk++

	c.exec()
}

func (c *Cpu) Log(a ...interface{}) {
	if c.verbose {
		fmt.Print(a...)
	}
}

func (c *Cpu) LogAf(align int, format string, a ...interface{}) {
	if c.verbose {
		n, _ := fmt.Printf(format, a...)
		if align > n {
			s := ""
			for i := n; i < align; i++ {
				s = fmt.Sprintf("%s ", s)
			}
			fmt.Printf(s)
		}
	}
}

func (c *Cpu) Logf(format string, a ...interface{}) {
	if c.verbose {
		fmt.Printf(format, a...)
	}
}

func (c *Cpu) nmi() {
	// do the other ones stay clear?? research this...
	c.interrupts &= 0xFE
	c._push16(c.rg.spc.pc.read())
	c.php()
	c.rg.spc.pc.write(c.read16(0xFFFA))
	c.inInt = true
	c.clk += 7
}

func (c *Cpu) exec() bool {

	// increment now or at the end?
	c.clk += 1

	switch c.interrupts {
	case cpuIntNMI:
		c.nmi()
	}

	c.curr.opr = c.fetch()
	opCode := c.curr.opr & 0xFF
	c.curr.ins = &c.ins[opCode]

	if c.curr.ins.opLength == 0 {
		c.Logf("Read 0x%02x - %s - which is an invalid instruction!\n", opCode, c.curr.ins.opName)
		c.rg.spc.pc.val += uint16(c.curr.ins.opLength)
		return false
	}

	c.LogAf(30, "0x%04x: 0x%02x - %s %s", c.rg.spc.pc.val, opCode, c.curr.ins.opName, c.getOperandString(c.curr.ins))
	c.curr.ins.eval()
	c.rg.spc.pc.val += uint16(c.curr.ins.opLength)

	c.Logf("%s", c.rg)

	if c.curr.ins.opName == "BRK" {
		// probably need to remove this...
		return false
	}
	return true
}

func (c *Cpu) fetch() uint32 {
	op0 := c.read8(c.rg.spc.pc.val)
	op1 := c.read8(c.rg.spc.pc.val + 1)
	op2 := c.read8(c.rg.spc.pc.val + 2)
	return uint32(op0) | uint32(op1)<<8 | uint32(op2)<<16
}

func (c *Cpu) brk() {
	if c.disableBreak {
		return
	}

	c.rg.spc.ps.set(bB, bB)
	// The BRK instruction forces the generation of an interrupt request.
	// The program counter and processor status are pushed on the stack
	// then the IRQ interrupt vector at $FFFE/F is loaded into the PC and the break flag
	// in the status set to one.

	c._push16(c.rg.spc.pc.read())
	c.php()
	c.sei()

	// needs more work, don't really understand it yet...
	c.rg.spc.pc.write(c.read16(0xFFFE))
}

func (c *Cpu) getOperandString(ins *Instruction) string {
	op1 := uint16(c.curr.opr&0xFF00) >> 8
	op12 := uint16((c.curr.opr & 0xFFFF00) >> 8)
	str := ""
	switch ins.addrMode {
	case ModeImplied:
	case ModeAccumulator:
	case ModeImmediate:
		str = fmt.Sprintf("#$%02x", op1)
	case ModeZeroPage:
		str = fmt.Sprintf("$%02x", op1)
	case ModeIndexedZeroPageX:
		str = fmt.Sprintf("$%02x, x", op1)
	case ModeIndexedZeroPageY:
		str = fmt.Sprintf("$%02x, y", op1)
	case ModeAbsolute:
		str = fmt.Sprintf("$%04x", op12)
	case ModeIndexedAbsoluteX:
		str = fmt.Sprintf("$%04x, x", op12)
	case ModeIndexedAbsoluteY:
		str = fmt.Sprintf("$%04x, y", op12)
	case ModeIndexedIndirectX:
		str = fmt.Sprintf("($%04x, x)", op12)
	case ModeIndirectIndexedY:
		str = fmt.Sprintf("($%04x, y)", op12)
	case ModeIndirect:
		str = fmt.Sprintf("($%04x)", op12)
	case ModeRelative:
		str = fmt.Sprintf("#$%02x", op1)
	case ModeInvalid:
		fallthrough
	default:
		panic(fmt.Sprintf("invalid address mode: %d", ins.addrMode))
	}
	return str
}

func (c *Cpu) getOperandAddr(ins *Instruction) uint16 {
	op1 := uint16(c.curr.opr&0xFF00) >> 8
	op12 := uint16((c.curr.opr & 0xFFFF00) >> 8)
	switch ins.addrMode {
	case ModeImmediate:
		return c.rg.spc.pc.read() + 1
	case ModeZeroPage:
		return op1
	case ModeIndexedZeroPageX:
		return (op1 + uint16(c.rg.gp.ix.x.read())) % 256
	case ModeIndexedZeroPageY:
		return (op1 + uint16(c.rg.gp.ix.y.read())) % 256
	case ModeAbsolute:
		return op12
	case ModeIndexedAbsoluteX:
		return op12 + uint16(c.rg.gp.ix.x.read())
	case ModeIndexedAbsoluteY:
		return op12 + uint16(c.rg.gp.ix.y.read())
	case ModeIndexedIndirectX:
		return c.read16(op1 + uint16(c.rg.gp.ix.x.read()))
	case ModeIndirectIndexedY:
		return c.read16(op1) + uint16(c.rg.gp.ix.y.read())
	case ModeIndirect:
		// http://www.obelisk.me.uk/6502/reference.html#JMP:
		// An original 6502 has does not correctly fetch the target address if the indirect vector falls on a page boundary
		// (e.g. $xxFF where xx is any value from $00 to $FF). In this case fetches the LSB from $xxFF as expected but takes
		// the MSB from $xx00. This is fixed in some later chips like the 65SC02 so for compatibility always ensure the
		// indirect vector is not at the end of the page.
		if op1 == 0xFF {
			l := uint16(c.read8(op12))
			h := uint16(c.read8(op12 & 0xFF00))
			return l | h<<8
		} else {
			return c.read16(op12)
		}
	case ModeRelative:
		// op1 -128,127 so we can jump backwards
		return c.rg.spc.pc.read() + uint16(int8(op1))
	case ModeInvalid:
		fallthrough
	default:
		panic("Invalid instruction address mode")
	}
}

// Move Commands:
func (c *Cpu) sta() {
	c.write8(c.getOperandAddr(c.curr.ins), c.rg.gp.ac.read())
}
func (c *Cpu) stx() {
	c.write8(c.getOperandAddr(c.curr.ins), c.rg.gp.ix.x.read())
}
func (c *Cpu) sty() {
	c.write8(c.getOperandAddr(c.curr.ins), c.rg.gp.ix.y.read())
}

func (c *Cpu) lda() {
	c.rg.gp.ac.write(c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) ldx() {
	c.rg.gp.ix.x.write(c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ix.x.read()))
}
func (c *Cpu) ldy() {
	c.rg.gp.ix.y.write(c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ix.y.read()))
}

func (c *Cpu) tax() {
	c.rg.gp.ix.x.write(c.rg.gp.ac.read())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ix.x.read()))
}
func (c *Cpu) tay() {
	c.rg.gp.ix.y.write(c.rg.gp.ac.read())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ix.y.read()))
}
func (c *Cpu) txa() {
	c.rg.gp.ac.write(c.rg.gp.ix.x.read())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) tya() {
	c.rg.gp.ac.write(c.rg.gp.ix.y.read())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}

func (c *Cpu) txs() {
	c.rg.spc.sp.write(c.rg.gp.ix.x.read())
}
func (c *Cpu) tsx() {
	c.rg.gp.ix.x.write(c.rg.spc.sp.read())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ix.x.read()))
}

func (c *Cpu) _push8(val uint8) {
	sp := c.rg.spc.sp.read()
	c.write8(uint16(sp)|0x100, val)
	c.rg.spc.sp.write(sp - 1)
}
func (c *Cpu) _push16(val uint16) {
	c._push8(uint8((val & 0xFF00) >> 8))
	c._push8(uint8(val & 0xFF))
}
func (c *Cpu) _pull8() uint8 {
	sp := c.rg.spc.sp.read() + 1
	c.rg.spc.sp.write(sp)
	return c.read8(uint16(sp) | 0x100)
}
func (c *Cpu) _pull16() uint16 {
	return uint16(c._pull8()) | uint16(c._pull8())<<8
}

func (c *Cpu) pha() {
	c._push8(c.rg.gp.ac.read())
}
func (c *Cpu) php() {
	c._push8(c.rg.spc.ps.read() | bB)
}

func (c *Cpu) pla() {
	c.rg.gp.ac.write(c._pull8())
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) plp() {
	c.rg.spc.ps.write(c._pull8())
}

// Jump/Flag Commands:
func (c *Cpu) bit() {
	mask := c.rg.gp.ac.read()
	value := c.read8(c.getOperandAddr(c.curr.ins))
	result := value & mask
	c.rg.spc.ps.set(bZ, int8(result))
	c.rg.spc.ps.set(bN|bV, int8(value))
}

func (c *Cpu) clc() {
	c.rg.spc.ps.set(bC, 0)
}
func (c *Cpu) sec() {
	c.rg.spc.ps.set(bC, bC)
}
func (c *Cpu) sed() {
	c.rg.spc.ps.set(bD, bD)
}
func (c *Cpu) cld() {
	c.rg.spc.ps.set(bD, 0)
}
func (c *Cpu) clv() {
	c.rg.spc.ps.set(bV, 0)
}
func (c *Cpu) sei() {
	c.rg.spc.ps.set(bI, bI)
}
func (c *Cpu) cli() {
	c.rg.spc.ps.set(bI, 0)
}

func (c *Cpu) jmp() {
	// take into account PC increment
	// might be better to move operand get to exec
	c.rg.spc.pc.write(c.getOperandAddr(c.curr.ins) - uint16(c.curr.ins.opLength))
}

func (c *Cpu) _branch(flag uint8, test uint8) {
	if (c.rg.spc.ps.read() & flag) == test {
		c.rg.spc.pc.write(c.getOperandAddr(c.curr.ins))
	}
}

func (c *Cpu) bpl() {
	c._branch(bN, 0)
}
func (c *Cpu) bmi() {
	c._branch(bN, bN)
}

func (c *Cpu) bvc() {
	c._branch(bV, 0)
}
func (c *Cpu) bvs() {
	c._branch(bV, bV)
}

func (c *Cpu) bcc() {
	c._branch(bC, 0)
}
func (c *Cpu) bcs() {
	c._branch(bC, bC)
}

func (c *Cpu) bne() {
	c._branch(bZ, 0)
}
func (c *Cpu) beq() {
	c._branch(bZ, bZ)
}

func (c *Cpu) jsr() {
	retAddr := c.rg.spc.pc.read() + uint16(c.curr.ins.opLength)
	c._push16(retAddr - 1)
	c.jmp()
}
func (c *Cpu) rts() {
	c.rg.spc.pc.write(c._pull16())
}

func (c *Cpu) rti() {
	c.plp()
	c.rg.spc.pc.write(c._pull16() - 1) // ?? -1??
	c.inInt = false
	c.clk += 7
}

func (c *Cpu) nop() {}

// Logical and arithmetic commands:
func (c *Cpu) ora() {
	c.rg.gp.ac.write(c.rg.gp.ac.read() | c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) and() {
	c.rg.gp.ac.write(c.rg.gp.ac.read() & c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) eor() {
	c.rg.gp.ac.write(c.rg.gp.ac.read() ^ c.read8(c.getOperandAddr(c.curr.ins)))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
}
func (c *Cpu) _add(opr uint8) {
	result := uint16(c.rg.gp.ac.read()) + uint16(opr) + uint16(c.rg.spc.ps.read()&bC)>>C
	if result > 0xFF {
		c.rg.spc.ps.set(bC, bC)
	} else {
		c.rg.spc.ps.set(bC, 0)
	}

	// signed overflows and underflow's - if the addends sign bits are equal and result sign differs
	// eg: 127 + 3 = 130 ( -126 )
	// eg: -10 -120 = -130 ( 2 )
	if ((c.rg.gp.ac.read()^opr)&0x80) == 0 && ((uint16(c.rg.gp.ac.read())^result)&0x80) != 0 {
		c.rg.spc.ps.set(bV, bV)
	} else {
		c.rg.spc.ps.set(bV, 0)
	}
	c.rg.gp.ac.write(uint8(result & 0xFF))
	c.rg.spc.ps.set(bZ|bN, int8(c.rg.gp.ac.read()))
	// oh and we're not handling decimal mode...
}

func (c *Cpu) adc() {
	c._add(c.read8(c.getOperandAddr(c.curr.ins)))
}
func (c *Cpu) sbc() {
	c._add(c.read8(c.getOperandAddr(c.curr.ins)) ^ 0xFF)
}

func (c *Cpu) _cmp(op1 uint8) {
	op2 := c.read8(c.getOperandAddr(c.curr.ins))
	r := int8(op1 - op2)

	if r >= 0 {
		c.rg.spc.ps.set(bC, bC)
	} else {
		c.rg.spc.ps.set(bC, 0)
	}
	c.rg.spc.ps.set(bZ|bN, r)
}

func (c *Cpu) cmp() {
	c._cmp(c.rg.gp.ac.read())
}

func (c *Cpu) cpx() {
	c._cmp(c.rg.gp.ix.x.read())
}

func (c *Cpu) cpy() {
	c._cmp(c.rg.gp.ix.y.read())
}

func (c *Cpu) dec() {
	v := c.read8(c.getOperandAddr(c.curr.ins)) - 1
	c.write8(c.getOperandAddr(c.curr.ins), v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) dex() {
	v := c.rg.gp.ix.x.read() - 1
	c.rg.gp.ix.x.write(v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) dey() {
	v := c.rg.gp.ix.y.read() - 1
	c.rg.gp.ix.y.write(v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) inc() {
	v := c.read8(c.getOperandAddr(c.curr.ins)) + 1
	c.write8(c.getOperandAddr(c.curr.ins), v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) inx() {
	v := c.rg.gp.ix.x.read() + 1
	c.rg.gp.ix.x.write(v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) iny() {
	v := c.rg.gp.ix.y.read() + 1
	c.rg.gp.ix.y.write(v)
	c.rg.spc.ps.set(bZ|bN, int8(v))
}

func (c *Cpu) asl() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.gp.ac.read()
		c.rg.spc.ps.set(bC, int8(v>>7)&bC)
		v <<= 1
		c.rg.gp.ac.write(v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	} else {
		v := c.read8(c.getOperandAddr(c.curr.ins))
		c.rg.spc.ps.set(bC, int8(v>>7)&bC)
		v <<= 1
		c.write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	}
}

func (c *Cpu) rol() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.gp.ac.read()
		fC := c.rg.spc.ps.read() & bC
		c.rg.spc.ps.set(bC, int8(v>>7)&bC)
		v = (v << 1) | fC
		c.rg.gp.ac.write(v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	} else {
		v := c.read8(c.getOperandAddr(c.curr.ins))
		fC := c.rg.spc.ps.read() & bC
		c.rg.spc.ps.set(bC, int8(v>>7)&bC)
		v = (v << 1) | fC
		c.write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	}
}

func (c *Cpu) lsr() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.gp.ac.read()
		c.rg.spc.ps.set(bC, int8(v)&bC)
		v >>= 1
		c.rg.gp.ac.write(v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	} else {
		v := c.read8(c.getOperandAddr(c.curr.ins))
		c.rg.spc.ps.set(bC, int8(v)&bC)
		v >>= 1
		c.write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	}
}

func (c *Cpu) ror() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.gp.ac.read()
		fC := c.rg.spc.ps.read() & bC
		c.rg.spc.ps.set(bC, int8(v)&bC)
		v = (v >> 1) | (fC << 7)
		c.rg.gp.ac.write(v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	} else {
		v := c.read8(c.getOperandAddr(c.curr.ins))
		fC := c.rg.spc.ps.read() & bC
		c.rg.spc.ps.set(bC, int8(v)&bC)
		v = (v >> 1) | (fC << 7)
		c.write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.spc.ps.set(bZ|bN, int8(v))
	}
}
