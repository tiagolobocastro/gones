package gones

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tiagolobocastro/gones/nes/common"
	"github.com/tiagolobocastro/gones/nes/cpu"
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
	pgX bool
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
	common.BusExtInt
	interrupt

	ins [256]Instruction

	curr Context

	rg cpu.Registers

	clk      int
	clkExtra int

	verbose      bool
	disableBreak bool

	inInt      bool
	interrupts uint8

	// a bit messy but ok for now
	f *os.File

	// internal buffer to make logging compatible with previous fmt.print*
	bufStr string
}

func (c *Cpu) init(busInt common.BusExtInt, verbose bool) {
	c.verbose = verbose
	c.disableBreak = true

	c.rg.Init()
	c.setupIns()

	c.BusExtInt = busInt

	if !c.verbose {
		// set log to stdout just in case we change it during debugging
		log.SetOutput(os.Stdout)
		return
	}

	f, err := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	c.f = f
	//wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(f)
}

func (c *Cpu) reset() {
	c.rg.Init()
	c.inInt = false
	c.rg.Spc.Pc.Write(c.Read16(0xFFFC))
	c.curr.ins = nil
}

func (c *Cpu) Log(a ...interface{}) {
	if c.verbose {
		log.Print(a...)
	}
}

func (c *Cpu) LogAf(align int, format string, a ...interface{}) {
	if c.verbose {
		s := fmt.Sprintf(format, a...)
		n := len(s)
		if align > n {
			for i := n; i < align; i++ {
				s = fmt.Sprintf("%s ", s)
			}
		}
		c.bufStr += s

		if strings.IndexByte(s, '\n') > 0 {
			log.Printf(c.bufStr)
			c.bufStr = ""
		}
	}
}

func (c *Cpu) Logf(format string, a ...interface{}) {
	if c.verbose {
		s := fmt.Sprintf(format, a...)
		c.bufStr += s

		if strings.IndexByte(s, '\n') > 0 {
			log.Print(c.bufStr)
			c.bufStr = ""
		}
	}
}

func (c *Cpu) nmi() {
	// do the other ones stay clear?? research this...
	c.interrupts &= 0xFE
	c._push16(c.rg.Spc.Pc.Read())
	c.php()
	c.rg.Spc.Pc.Write(c.Read16(0xFFFA))
	c.inInt = true
	c.clk += 7
}

func (c *Cpu) tick() int {

	clk := c.clk
	c.exec()
	ticks := c.clk - clk
	return ticks
}

func (c *Cpu) exec() {

	switch c.interrupts {
	case cpuIntNMI:
		c.nmi()
		c.clk += 7
	}

	c.curr.pgX = false
	c.curr.opr = c.fetch()
	opCode := c.curr.opr & 0xFF
	c.curr.ins = &c.ins[opCode]

	if c.curr.ins.opLength == 0 {
		c.Logf("Read 0x%02x - %s - which is an invalid instruction!\n", opCode, c.curr.ins.opName)
		c.rg.Spc.Pc.Val += uint16(c.curr.ins.opLength)

		// testing
		panic(fmt.Errorf("invalid instruction, opcode: 0x%02x", opCode))

		return
	}

	if c.verbose {
		c.LogAf(30, "0x%04x: 0x%02x - %s %s", c.rg.Spc.Pc.Val, opCode, c.curr.ins.opName, c.getOperandString(c.curr.ins))
	}

	c.curr.ins.eval()
	c.rg.Spc.Pc.Val += uint16(c.curr.ins.opLength)

	if c.verbose {
		c.Logf("%s\n", c.rg)
	}

	if c.curr.ins.opName == "BRK" {
		// probably need to remove this...
		return
	}

	c.clk += int(c.curr.ins.opCycles)
	if c.curr.pgX {
		c.clk += int(c.curr.ins.opPageCycles)
	}
}

func (c *Cpu) fetch() uint32 {
	op01 := c.Read16(c.rg.Spc.Pc.Val)
	op2 := c.Read8(c.rg.Spc.Pc.Val + 2)
	return uint32(op01) | uint32(op2)<<16
}

func (c *Cpu) brk() {

	if c.disableBreak {
		return
	}

	c.rg.Spc.Ps.Set(cpu.BB, cpu.BB)
	// The BRK instruction forces the generation of an interrupt request.
	// The program counter and processor status are pushed on the stack
	// then the IRQ interrupt vector at $FFFE/F is loaded into the PC and the break flag
	// in the status set to one.

	c._push16(c.rg.Spc.Pc.Read())
	c.php()
	c.sei()

	// needs more work, don't really understand it yet...
	c.rg.Spc.Pc.Write(c.Read16(0xFFFE))
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
		str = fmt.Sprintf("$%02x, X", op1)
	case ModeIndexedZeroPageY:
		str = fmt.Sprintf("$%02x, Y", op1)
	case ModeAbsolute:
		str = fmt.Sprintf("$%04x", op12)
	case ModeIndexedAbsoluteX:
		str = fmt.Sprintf("$%04x, X", op12)
	case ModeIndexedAbsoluteY:
		str = fmt.Sprintf("$%04x, Y", op12)
	case ModeIndexedIndirectX:
		str = fmt.Sprintf("($%04x, X)", op12)
	case ModeIndirectIndexedY:
		str = fmt.Sprintf("($%04x, Y)", op12)
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

func pageCrossed(a, b uint16) bool {
	return a&0xFF00 != b&0xFF00
}

func (c *Cpu) getOperandAddr(ins *Instruction) uint16 {
	op1 := uint16(c.curr.opr&0xFF00) >> 8
	op12 := uint16((c.curr.opr & 0xFFFF00) >> 8)
	switch ins.addrMode {
	case ModeImmediate:
		return c.rg.Spc.Pc.Read() + 1
	case ModeZeroPage:
		return op1
	case ModeIndexedZeroPageX:
		return (op1 + uint16(c.rg.Gp.Ix.X.Read())) % 256
	case ModeIndexedZeroPageY:
		return (op1 + uint16(c.rg.Gp.Ix.Y.Read())) % 256
	case ModeAbsolute:
		return op12
	case ModeIndexedAbsoluteX:
		x := uint16(c.rg.Gp.Ix.X.Read())
		addr := op12 + x
		c.curr.pgX = pageCrossed(addr-x, addr)
		return addr
	case ModeIndexedAbsoluteY:
		y := uint16(c.rg.Gp.Ix.Y.Read())
		addr := op12 + y
		c.curr.pgX = pageCrossed(addr-y, addr)
		return addr
	case ModeIndexedIndirectX:
		return c.Read16(op1 + uint16(c.rg.Gp.Ix.X.Read()))
	case ModeIndirectIndexedY:
		y := uint16(c.rg.Gp.Ix.Y.Read())
		addr := c.Read16(op1) + y
		c.curr.pgX = pageCrossed(addr-y, addr)
		return addr
	case ModeIndirect:
		// http://www.obelisk.me.uk/6502/reference.html#JMP:
		// An original 6502 has does not correctly fetch the target address if the indirect vector falls on a page boundary
		// (e.g. $xxFF where xx is any value from $00 to $FF). In this case fetches the LSB from $xxFF as expected but takes
		// the MSB from $xx00. This is fixed in some later chips like the 65SC02 so for compatibility always ensure the
		// indirect vector is not at the end of the page.
		if op1 == 0xFF {
			l := uint16(c.Read8(op12))
			h := uint16(c.Read8(op12 & 0xFF00))
			return l | h<<8
		} else {
			return c.Read16(op12)
		}
	case ModeRelative:
		// op1 -128,127 so we can jump backwards
		return c.rg.Spc.Pc.Read() + uint16(int8(op1))
	case ModeInvalid:
		fallthrough
	default:
		panic(fmt.Errorf("invalid instruction address mode: %d", ins.addrMode))
	}
}

// Move Commands:
func (c *Cpu) sta() {
	c.Write8(c.getOperandAddr(c.curr.ins), c.rg.Gp.Ac.Read())
}
func (c *Cpu) stx() {
	c.Write8(c.getOperandAddr(c.curr.ins), c.rg.Gp.Ix.X.Read())
}
func (c *Cpu) sty() {
	c.Write8(c.getOperandAddr(c.curr.ins), c.rg.Gp.Ix.Y.Read())
}

func (c *Cpu) lda() {
	c.rg.Gp.Ac.Write(c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) ldx() {
	c.rg.Gp.Ix.X.Write(c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ix.X.Read()))
}
func (c *Cpu) ldy() {
	c.rg.Gp.Ix.Y.Write(c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ix.Y.Read()))
}

func (c *Cpu) tax() {
	c.rg.Gp.Ix.X.Write(c.rg.Gp.Ac.Read())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ix.X.Read()))
}
func (c *Cpu) tay() {
	c.rg.Gp.Ix.Y.Write(c.rg.Gp.Ac.Read())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ix.Y.Read()))
}
func (c *Cpu) txa() {
	c.rg.Gp.Ac.Write(c.rg.Gp.Ix.X.Read())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) tya() {
	c.rg.Gp.Ac.Write(c.rg.Gp.Ix.Y.Read())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}

func (c *Cpu) txs() {
	c.rg.Spc.Sp.Write(c.rg.Gp.Ix.X.Read())
}
func (c *Cpu) tsx() {
	c.rg.Gp.Ix.X.Write(c.rg.Spc.Sp.Read())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ix.X.Read()))
}

func (c *Cpu) _push8(val uint8) {
	sp := c.rg.Spc.Sp.Read()
	c.Write8(uint16(sp)|0x100, val)
	c.rg.Spc.Sp.Write(sp - 1)
}
func (c *Cpu) _push16(val uint16) {
	c._push8(uint8((val & 0xFF00) >> 8))
	c._push8(uint8(val & 0xFF))
}
func (c *Cpu) _pull8() uint8 {
	sp := c.rg.Spc.Sp.Read() + 1
	c.rg.Spc.Sp.Write(sp)
	return c.Read8(uint16(sp) | 0x100)
}
func (c *Cpu) _pull16() uint16 {
	return uint16(c._pull8()) | uint16(c._pull8())<<8
}

func (c *Cpu) pha() {
	c._push8(c.rg.Gp.Ac.Read())
}
func (c *Cpu) php() {
	c._push8(c.rg.Spc.Ps.Read() | cpu.BB)
}

func (c *Cpu) pla() {
	c.rg.Gp.Ac.Write(c._pull8())
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) plp() {
	c.rg.Spc.Ps.Write(c._pull8())
}

// Jump/Flag Commands:
func (c *Cpu) bit() {
	mask := c.rg.Gp.Ac.Read()
	value := c.Read8(c.getOperandAddr(c.curr.ins))
	result := value & mask
	c.rg.Spc.Ps.Set(cpu.BZ, int8(result))
	c.rg.Spc.Ps.Set(cpu.BN|cpu.BV, int8(value))
}

func (c *Cpu) clc() {
	c.rg.Spc.Ps.Set(cpu.BC, 0)
}
func (c *Cpu) sec() {
	c.rg.Spc.Ps.Set(cpu.BC, cpu.BC)
}
func (c *Cpu) sed() {
	c.rg.Spc.Ps.Set(cpu.BD, cpu.BD)
}
func (c *Cpu) cld() {
	c.rg.Spc.Ps.Set(cpu.BD, 0)
}
func (c *Cpu) clv() {
	c.rg.Spc.Ps.Set(cpu.BV, 0)
}
func (c *Cpu) sei() {
	c.rg.Spc.Ps.Set(cpu.BI, cpu.BI)
}
func (c *Cpu) cli() {
	c.rg.Spc.Ps.Set(cpu.BI, 0)
}

// branching requires more cycles
func (c *Cpu) addBranchCycles(addr uint16) {
	c.clk++
	pc := c.rg.Spc.Pc.Val + uint16(c.curr.ins.opLength)
	if pageCrossed(pc, addr) {
		c.clk++
	}
}

func (c *Cpu) jmp() {
	// take into account PC increment
	// might be better to move operand get to exec
	addr := c.getOperandAddr(c.curr.ins) - uint16(c.curr.ins.opLength)
	c.rg.Spc.Pc.Write(addr)
}

func (c *Cpu) _branch(flag uint8, test uint8) {
	if (c.rg.Spc.Ps.Read() & flag) == test {
		addr := c.getOperandAddr(c.curr.ins)
		c.addBranchCycles(addr)
		c.rg.Spc.Pc.Write(addr)
	}
}

func (c *Cpu) bpl() {
	c._branch(cpu.BN, 0)
}
func (c *Cpu) bmi() {
	c._branch(cpu.BN, cpu.BN)
}

func (c *Cpu) bvc() {
	c._branch(cpu.BV, 0)
}
func (c *Cpu) bvs() {
	c._branch(cpu.BV, cpu.BV)
}

func (c *Cpu) bcc() {
	c._branch(cpu.BC, 0)
}
func (c *Cpu) bcs() {
	c._branch(cpu.BC, cpu.BC)
}

func (c *Cpu) bne() {
	c._branch(cpu.BZ, 0)
}
func (c *Cpu) beq() {
	c._branch(cpu.BZ, cpu.BZ)
}

func (c *Cpu) jsr() {
	retAddr := c.rg.Spc.Pc.Read() + uint16(c.curr.ins.opLength)
	c._push16(retAddr - 1)
	c.jmp()
}
func (c *Cpu) rts() {
	c.rg.Spc.Pc.Write(c._pull16())
}

func (c *Cpu) rti() {
	c.plp()
	c.rg.Spc.Pc.Write(c._pull16() - 1) // ?? -1??
	c.inInt = false
	c.clk += 7
}

func (c *Cpu) nop() {}

// Logical and arithmetic commands:
func (c *Cpu) ora() {
	c.rg.Gp.Ac.Write(c.rg.Gp.Ac.Read() | c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) and() {
	c.rg.Gp.Ac.Write(c.rg.Gp.Ac.Read() & c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) eor() {
	c.rg.Gp.Ac.Write(c.rg.Gp.Ac.Read() ^ c.Read8(c.getOperandAddr(c.curr.ins)))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
}
func (c *Cpu) _add(opr uint8) {
	result := uint16(c.rg.Gp.Ac.Read()) + uint16(opr) + uint16(c.rg.Spc.Ps.Read()&cpu.BC)>>cpu.C
	if result > 0xFF {
		c.rg.Spc.Ps.Set(cpu.BC, cpu.BC)
	} else {
		c.rg.Spc.Ps.Set(cpu.BC, 0)
	}

	// signed overflows and underflow's - if the addends sign bits are equal and result sign differs
	// eg: 127 + 3 = 130 ( -126 )
	// eg: -10 -120 = -130 ( 2 )
	if ((c.rg.Gp.Ac.Read()^opr)&0x80) == 0 && ((uint16(c.rg.Gp.Ac.Read())^result)&0x80) != 0 {
		c.rg.Spc.Ps.Set(cpu.BV, cpu.BV)
	} else {
		c.rg.Spc.Ps.Set(cpu.BV, 0)
	}
	c.rg.Gp.Ac.Write(uint8(result & 0xFF))
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(c.rg.Gp.Ac.Read()))
	// oh and we're not handling decimal mode...
}

func (c *Cpu) adc() {
	c._add(c.Read8(c.getOperandAddr(c.curr.ins)))
}
func (c *Cpu) sbc() {
	c._add(c.Read8(c.getOperandAddr(c.curr.ins)) ^ 0xFF)
}

func (c *Cpu) _cmp(op1 uint8) {
	op2 := c.Read8(c.getOperandAddr(c.curr.ins))
	r := int8(op1 - op2)

	if op1 >= op2 {
		c.rg.Spc.Ps.Set(cpu.BC, cpu.BC)
	} else {
		c.rg.Spc.Ps.Set(cpu.BC, 0)
	}
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, r)
}

func (c *Cpu) cmp() {
	c._cmp(c.rg.Gp.Ac.Read())
}

func (c *Cpu) cpx() {
	c._cmp(c.rg.Gp.Ix.X.Read())
}

func (c *Cpu) cpy() {
	c._cmp(c.rg.Gp.Ix.Y.Read())
}

func (c *Cpu) dec() {
	v := c.Read8(c.getOperandAddr(c.curr.ins)) - 1
	c.Write8(c.getOperandAddr(c.curr.ins), v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) dex() {
	v := c.rg.Gp.Ix.X.Read() - 1
	c.rg.Gp.Ix.X.Write(v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) dey() {
	v := c.rg.Gp.Ix.Y.Read() - 1
	c.rg.Gp.Ix.Y.Write(v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) inc() {
	v := c.Read8(c.getOperandAddr(c.curr.ins)) + 1
	c.Write8(c.getOperandAddr(c.curr.ins), v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) inx() {
	v := c.rg.Gp.Ix.X.Read() + 1
	c.rg.Gp.Ix.X.Write(v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) iny() {
	v := c.rg.Gp.Ix.Y.Read() + 1
	c.rg.Gp.Ix.Y.Write(v)
	c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
}

func (c *Cpu) asl() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.Gp.Ac.Read()
		c.rg.Spc.Ps.Set(cpu.BC, int8(v>>7)&cpu.BC)
		v <<= 1
		c.rg.Gp.Ac.Write(v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	} else {
		v := c.Read8(c.getOperandAddr(c.curr.ins))
		c.rg.Spc.Ps.Set(cpu.BC, int8(v>>7)&cpu.BC)
		v <<= 1
		c.Write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	}
}

func (c *Cpu) rol() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.Gp.Ac.Read()
		fC := c.rg.Spc.Ps.Read() & cpu.BC
		c.rg.Spc.Ps.Set(cpu.BC, int8(v>>7)&cpu.BC)
		v = (v << 1) | fC
		c.rg.Gp.Ac.Write(v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	} else {
		v := c.Read8(c.getOperandAddr(c.curr.ins))
		fC := c.rg.Spc.Ps.Read() & cpu.BC
		c.rg.Spc.Ps.Set(cpu.BC, int8(v>>7)&cpu.BC)
		v = (v << 1) | fC
		c.Write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	}
}

func (c *Cpu) lsr() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.Gp.Ac.Read()
		c.rg.Spc.Ps.Set(cpu.BC, int8(v)&cpu.BC)
		v >>= 1
		c.rg.Gp.Ac.Write(v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	} else {
		v := c.Read8(c.getOperandAddr(c.curr.ins))
		c.rg.Spc.Ps.Set(cpu.BC, int8(v)&cpu.BC)
		v >>= 1
		c.Write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	}
}

func (c *Cpu) ror() {
	if c.curr.ins.addrMode == ModeAccumulator {
		v := c.rg.Gp.Ac.Read()
		fC := c.rg.Spc.Ps.Read() & cpu.BC
		c.rg.Spc.Ps.Set(cpu.BC, int8(v)&cpu.BC)
		v = (v >> 1) | (fC << 7)
		c.rg.Gp.Ac.Write(v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	} else {
		v := c.Read8(c.getOperandAddr(c.curr.ins))
		fC := c.rg.Spc.Ps.Read() & cpu.BC
		c.rg.Spc.Ps.Set(cpu.BC, int8(v)&cpu.BC)
		v = (v >> 1) | (fC << 7)
		c.Write8(c.getOperandAddr(c.curr.ins), v)
		c.rg.Spc.Ps.Set(cpu.BZ|cpu.BN, int8(v))
	}
}
