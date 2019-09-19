package gones

import (
	"fmt"
	"io"
	"strings"
)

func (n *nes) init(cartPath string) {
	n.bus.init()

	if n.cart.init(cartPath) != nil {
		panic("ups...")
	}

	n.ram.init(0x2000)
	n.vRam.init(0x2000)
	n.fakeRam.init(0x7)

	n.cpu.init(n.bus.getBusInt(MapCPUId), n.verbose)
	n.ppu.init(n.bus.getBusInt(MapPPUId), n.verbose)
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
