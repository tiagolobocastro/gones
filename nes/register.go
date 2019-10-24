package gones

import (
	"fmt"
	"log"
)

const (
	C = 0 // Carry
	Z = 1 // Zero Result
	I = 2 // Interrupt Disable
	D = 3 // Decimal Mode
	B = 4 // Break Command
	E = 5 // Expansion
	V = 6 // Overflow
	N = 7 // Negative Result

	bC = 1 << C
	bZ = 1 << Z
	bI = 1 << I
	bD = 1 << D
	bB = 1 << B
	bE = 1 << E
	bV = 1 << V
	bN = 1 << N
)

func (psr *ps_register) read() uint8 {
	return 0 |
		psr.bit[C]<<C |
		psr.bit[Z]<<Z |
		psr.bit[I]<<I |
		psr.bit[D]<<D |
		psr.bit[B]<<B |
		psr.bit[E]<<E |
		psr.bit[V]<<V |
		psr.bit[N]<<N
}

func (psr *ps_register) set(flags int, value int8) {

	if (flags & bC) == bC {
		if value&bC == bC {
			psr.bit[C] = 1
		} else {
			psr.bit[C] = 0
		}
	}
	if (flags & bD) == bD {
		if value&bD == bD {
			psr.bit[D] = 1
		} else {
			psr.bit[D] = 0
		}
	}
	if (flags & bZ) == bZ {
		if value == 0 {
			psr.bit[Z] = 1
		} else {
			psr.bit[Z] = 0
		}
	}
	if (flags & bN) == bN {
		if value < 0 {
			psr.bit[N] = 1
		} else {
			psr.bit[N] = 0
		}
	}
	if (flags & bB) == bB {
		if value&bB == bB {
			psr.bit[B] = 1
		} else {
			psr.bit[B] = 0
		}
	}
	if (flags & bI) == bI {
		if value&bI == bI {
			psr.bit[I] = 1
		} else {
			psr.bit[I] = 0
		}
	}
	if (flags & bV) == bV {
		if value&bV == bV {
			psr.bit[V] = 1
		} else {
			psr.bit[V] = 0
		}
	}
}

func (psr *ps_register) write(value uint8) {
	psr.bit[C] = (value >> C) & 1
	psr.bit[Z] = (value >> Z) & 1
	psr.bit[I] = (value >> I) & 1
	psr.bit[D] = (value >> D) & 1
	psr.bit[B] = (value >> B) & 1
	psr.bit[E] = (value >> E) & 1
	psr.bit[V] = (value >> V) & 1
	psr.bit[N] = (value >> N) & 1
}

func (psr ps_register) String() string {
	return fmt.Sprintf("%s: 0x%02x (N:%d V:%d E:%d B:%d D:%d I:%d Z:%d C:%d)", psr.name, psr.read(),
		psr.bit[N], psr.bit[V], psr.bit[E], psr.bit[B], psr.bit[D], psr.bit[I], psr.bit[Z], psr.bit[C])
}

func (psr ps_register) String2() string {
	return fmt.Sprintf("%s: 0x%02x", psr.name, psr.read())
}

func (psr *ps_register) init(name string, val uint8) {
	psr.write(val)
	psr.name = name
}

func (r register) String() string {
	return fmt.Sprintf("%s: 0x%02x", r.name, r.val)
}
func (r *register) init(name string, val uint8) {
	r.val = val
	r.name = name
}
func (r *register) initx(name string, val uint8, onWrite func(), onRead func() uint8) {
	r.init(name, val)
	r.onWrite = onWrite
	r.onRead = onRead
}
func (r *register) set(w uint8) {
	r.val |= w
}
func (r *register) clr(w uint8) {
	r.val &= w ^ 0xFF
}

func (r *register) write(w uint8) {
	r.val = w

	if r.onWrite != nil {
		r.onWrite()
	}
}
func (r *register) read() uint8 {
	// add logging so we can debug it, same to write actually
	// where to control logging level without having to propagate a flag to each component,
	// have a package level?? probably ok
	if r.onRead != nil {
		return r.onRead()
	}
	return r.val
}

func (r register16) String() string {
	return fmt.Sprintf("%s: 0x%04x", r.name, r.val)
}
func (r *register16) init(name string, val uint16) {
	r.val = val
	r.name = name
}
func (r *register16) write(w uint16) {
	r.val = w
}
func (r *register16) read() uint16 {
	return r.val
}

func (r *spc_registers) init(name string) {
	r.pc.init("pc", 0xFFFC)
	r.sp.init("sp", 0xFF)
	r.ps.init("ps", bB|bI|bZ|bE)
	r.name = name
}
func (r spc_registers) String() string {
	return fmt.Sprintf("%s, %s, %s", r.pc, r.sp, r.ps)
}

func (r *ix_registers) init(name string, valx uint8, valy uint8) {
	r.x.init("x", valx)
	r.y.init("y", valy)
	r.name = name
}
func (r ix_registers) String() string {
	return fmt.Sprintf("%s, %s", r.x, r.y)
}

func (r *gp_registers) init(name string) {
	r.ac.init("ac", 0)
	r.ix.init("ix", 0, 0)
	r.name = name
}
func (r gp_registers) String() string {
	return fmt.Sprintf("%s, %s", r.ac, r.ix)
}

func (r *registers) init() {
	r.spc.init("spcr")
	r.gp.init("gpr")
}

func (r registers) print() {
	log.Println(r)
}
func (r registers) String() string {
	return fmt.Sprintf("%s, %s", r.spc, r.gp)
}
