package gones

import (
	"fmt"
	"io"
	"strings"
)

// CPU Mapping Table
// Address range 	Size 	Device
// $0000-$07FF 	$0800 	2KB internal RAM
// $0800-$0FFF 	$0800 	Mirrors of $0000-$07FF
// $1000-$17FF 	$0800
// $1800-$1FFF 	$0800
// $2000-$2007 	$0008 	NES PPU registers
// $2008-$3FFF 	$1FF8 	Mirrors of $2000-2007 (repeats every 8 bytes)
// $4000-$4017 	$0018 	NES APU and I/O registers
// $4018-$401F 	$0008 	APU and I/O functionality that is normally disabled. See CPU Test Mode.
// $4020-$FFFF 	$BFE0 	Cartridge space: PRG ROM, PRG RAM, and mapper registers (See Note)
var cpuMappingTable = []memMapEntry{
	{addrRange{0x0000, 0x07FF, 3}, devIdRAM},
	{addrRange{0x2000, 0x2007, 0x3FF}, devIdPPU},
	{addrRange{0x4000, 0x4017, 0}, devIdAPU},
	// size should be 8 but then clashes with the next, wtf??
	{addrRange{0x4018, 0x401F, 0}, devIdAPUIO},

	// probably need to break this down...
	// this is where the mapper comes into play?
	{addrRange{0x4020, 0xFFFF, 0}, devIdCART},
}

func (n *nes) init(cartPath string) {
	n.bus.init()

	if n.cart.init(cartPath) != nil {
		panic("ups...")
	}

	n.ram.init(0x2000)
	n.fakeRam.init(0x7)

	n.bus.connect(MapCPUId, n.getDevMapEntry(devIdRAM, &n.ram))
	n.bus.connect(MapCPUId, n.getDevMapEntry(devIdPPU, &n.fakeRam))

	// connect the cart into the bus only within the CPU devIdCart range
	n.bus.connectMap(MapCPUId, n.getDevMapEntry(devIdCART, nil).addrRange, n.cart.getMappingTable())

	n.cpu.init(n.bus.getBusInt(MapCPUId), n.verbose)
}

func (n *nes) getDevMapEntry(devId int, devInt busInt) *busMemMapEntry {
	for _, entry := range cpuMappingTable {
		if entry.deviceId == devId {
			return &busMemMapEntry{
				busAddrRange: busAddrRange{
					addrRange: entry.addrRange,
					size:      entry.end - entry.base,
				},
				busInt: devInt,
			}
		}
	}
	return nil
}

// from hexd from: https://skilldrick.github.io/easy6502/, eg:
//var easy6502Code string =  `0600: a9 01 85 02 a9 cc 8d 00 01 a9 01 aa a1 00 00 00
// 							0610: a9 05 aa 8e 00 02 a9 05 8d 01 02 a9 08 8d 02 02`

func (n *nes) loadEasyCode(code string) {

	findAddr := true
	for _, line := range strings.Split(strings.TrimSuffix(code, "\n"), "\n") {
		addr := 0
		var bt [16]int
		ns, err := fmt.Sscanf(line, "%x: %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x %x ",
			&addr, &bt[0], &bt[1], &bt[2], &bt[3], &bt[4], &bt[5], &bt[6], &bt[7],
			&bt[8], &bt[9], &bt[10], &bt[11], &bt[12], &bt[13], &bt[14], &bt[15])
		if err != nil && err != io.EOF {
			fmt.Printf("N: %x, E: %+v\n", ns, err)
		}

		if findAddr {
			findAddr = false
			n.cpu.rg.spc.pc.val = uint16(addr)
		}

		for _, b := range bt {
			n.ram.ram[addr] = byte(b)
			addr += 1
		}
	}
}

func (n *nes) stats() {
	nValid := 0
	nTotal := 0
	nImp := 0
	for _, in := range n.cpu.ins {
		if in.opName == "" {
			continue
		}

		nTotal += 1
		if in.opLength > 0 {
			nValid += 1

			if in.implemented {
				nImp += 1
			}
		}
	}
	fmt.Printf("\nTotal instructions: %d\nValid instructions: %d\nImplemented instructions: %d\nRemainingValid: %d\n", nTotal, nValid, nImp, nValid-nImp)
}

func (n *nes) Run() {
	for n.cpu.exec() {
	}
}

func (n *nes) reset() {
	n.init("")
}

func NewNES(verbose bool, cart string) *nes {
	nes := nes{verbose: verbose}
	nes.init(cart)
	return &nes
}
