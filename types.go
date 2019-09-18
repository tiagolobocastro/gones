package gones

type apu struct {
}

type register struct {
	val uint8

	name string
}

type register16 struct {
	val uint16

	name string
}

type ps_register struct {
	bit [8]byte

	name string
}

type spc_registers struct {
	pc register16
	sp register
	ps ps_register

	name string
}

type ix_registers struct {
	x register
	y register

	name string
}

type gp_registers struct {
	ac register
	ix ix_registers

	name string
}

type registers struct {
	spc     spc_registers
	gp      gp_registers
	verbose bool
}

const (
	// allows for validity test
	ModeInvalid = iota
	ModeZeroPage
	ModeIndexedZeroPageX
	ModeIndexedZeroPageY
	ModeAbsolute
	ModeIndexedAbsoluteX
	ModeIndexedAbsoluteY
	ModeIndirect
	ModeImplied
	ModeAccumulator
	ModeImmediate
	ModeRelative
	ModeIndexedIndirectX
	ModeIndirectIndexedY
)

type ppu struct {
	busInt
	*bus
}

type nes struct {
	bus

	cpu     Cpu
	ram     ram
	cart    Cartridge
	fakeRam ram
	ppu     ppu

	verbose bool
}

//  deviceId
const (
	devIdRAM = iota
	devIdPPU
	devIdAPU
	devIdAPUIO
	devIdCART
)

const (
	MapCPUId = iota
	//MapPPUId
)
