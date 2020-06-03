package cpu

import (
	"fmt"
	"log"

	"github.com/tiagolobocastro/gones/nes/common"
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

	BC = 1 << C
	BZ = 1 << Z
	BI = 1 << I
	BD = 1 << D
	BB = 1 << B
	BE = 1 << E
	BV = 1 << V
	BN = 1 << N
)

type ps_register struct {
	bit [8]byte

	name string
}

type spc_registers struct {
	Pc common.Register16
	Sp common.Register
	Ps ps_register

	name string
}

type ix_registers struct {
	X common.Register
	Y common.Register

	name string
}

type gp_registers struct {
	Ac common.Register
	Ix ix_registers

	name string
}

type Registers struct {
	Spc     spc_registers
	Gp      gp_registers
	verbose bool
}

func (psr *ps_register) Read() uint8 {
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

func (psr *ps_register) Set(flags int, value int8) {

	if (flags & BC) == BC {
		if value&BC == BC {
			psr.bit[C] = 1
		} else {
			psr.bit[C] = 0
		}
	}
	if (flags & BD) == BD {
		if value&BD == BD {
			psr.bit[D] = 1
		} else {
			psr.bit[D] = 0
		}
	}
	if (flags & BZ) == BZ {
		if value == 0 {
			psr.bit[Z] = 1
		} else {
			psr.bit[Z] = 0
		}
	}
	if (flags & BN) == BN {
		if value < 0 {
			psr.bit[N] = 1
		} else {
			psr.bit[N] = 0
		}
	}
	if (flags & BB) == BB {
		if value&BB == BB {
			psr.bit[B] = 1
		} else {
			psr.bit[B] = 0
		}
	}
	if (flags & BI) == BI {
		if value&BI == BI {
			psr.bit[I] = 1
		} else {
			psr.bit[I] = 0
		}
	}
	if (flags & BV) == BV {
		if value&BV == BV {
			psr.bit[V] = 1
		} else {
			psr.bit[V] = 0
		}
	}
}

func (psr *ps_register) Write(value uint8) {
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
	return fmt.Sprintf("%s: 0x%02x (N:%d V:%d E:%d B:%d D:%d I:%d Z:%d C:%d)", psr.name, psr.Read(),
		psr.bit[N], psr.bit[V], psr.bit[E], psr.bit[B], psr.bit[D], psr.bit[I], psr.bit[Z], psr.bit[C])
}

func (psr ps_register) String2() string {
	return fmt.Sprintf("%s: 0x%02x", psr.name, psr.Read())
}

func (psr *ps_register) init(name string, val uint8) {
	psr.Write(val)
	psr.name = name
}

func (r *spc_registers) init(name string) {
	r.Pc.Init("Pc", 0xFFFC)
	r.Sp.Init("Sp", 0xFF)
	r.Ps.init("Ps", BB|BI|BZ|BE)
	r.name = name
}
func (r spc_registers) String() string {
	return fmt.Sprintf("%s, %s, %s", r.Pc, r.Sp, r.Ps)
}

func (r *ix_registers) init(name string, valx uint8, valy uint8) {
	r.X.Init("X", valx)
	r.Y.Init("Y", valy)
	r.name = name
}
func (r ix_registers) String() string {
	return fmt.Sprintf("%s, %s", r.X, r.Y)
}

func (r *gp_registers) init(name string) {
	r.Ac.Init("Ac", 0)
	r.Ix.init("Ix", 0, 0)
	r.name = name
}
func (r gp_registers) String() string {
	return fmt.Sprintf("%s, %s", r.Ac, r.Ix)
}
func (r *Registers) Init() {
	r.Spc.init("spcr")
	r.Gp.init("gpr")
}

func (r Registers) print() {
	log.Println(r)
}
func (r Registers) String() string {
	return fmt.Sprintf("%s, %s", r.Spc, r.Gp)
}
