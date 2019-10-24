package gones

import (
	"image/color"
	"log"
)

func (n *nes) init(cartPath string) {
	n.bus.init()

	if n.cart.init(cartPath) != nil {
		panic("ups...")
	}

	n.ram.init(0x800)
	n.vRam.init(0x800)

	n.cpu.init(n.bus.getBusInt(MapCPUId), n.verbose)
	n.ppu.init(n.bus.getBusInt(MapPPUId), n.verbose, &n.cpu)
	n.dma.init(n.bus.getBusInt(MapDMAId))

	n.ppu.frameBuffer = make([]color.RGBA, 256*240)

	n.bus.connect(MapCPUId, &cpuMapper{n})
	n.bus.connect(MapPPUId, &ppuMapper{n})
	n.bus.connect(MapDMAId, &dmaMapper{n})

	n.cpu.reset()

	n.screen.init(n)
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
	log.Printf("\nTotal instructions: %d\nValid instructions: %d\nImplemented instructions: %d\nRemainingValid: %d\n", nTotal, nValid, nImp, nValid-nImp)
}

func (n *nes) Run() {
	for {

		if !n.dma.active() {
			// cpu stalled whilst dma is active
			n.cpu.tick()
		}

		n.dma.tick()

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3)

		if n.cpu.curr.ins.opName == "BRK" {
			break
		}
	}
}

func (n *nes) reset() {
	n.cpu.reset()
}

func NewNES(verbose bool, cart string) *nes {
	nes := nes{verbose: verbose}
	nes.init(cart)
	return &nes
}
