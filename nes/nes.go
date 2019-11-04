package gones

import (
	"log"
	"time"
)

func (n *nes) init(cartPath string) {
	n.bus.init()

	if n.cart.init(cartPath) != nil {
		panic("ups...")
	}

	n.ram.init(0x800)
	n.vRam.init(0x800)
	n.screen.init(n)

	n.cpu.init(n.bus.getBusInt(MapCPUId), n.verbose)
	n.ppu.init(n.bus.getBusInt(MapPPUId), n.verbose, &n.cpu, n.screen.pix.Pix)
	n.dma.init(n.bus.getBusInt(MapDMAId))

	n.bus.connect(MapCPUId, &cpuMapper{n})
	n.bus.connect(MapPPUId, &ppuMapper{n})
	n.bus.connect(MapDMAId, &dmaMapper{n})

	n.cpu.reset()
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

func (n *nes) Step(seconds float64) {
	cyclesPerSecond := float64(1790000)
	cyclesPerSecond *= seconds
	runCycles := int(cyclesPerSecond)

	ticks := 1
	extra := true

	for runCycles > 0 {
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			n.cpu.tick()
		} else {
			n.cpu.clkExtra = 1
		}

		ticks = 1
		if extra {
			ticks = n.cpu.clkExtra
			n.cpu.clkExtra = 0
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3 * ticks)
		n.dma.ticks(ticks)

		runCycles -= ticks
	}
}

func (n *nes) Run2() {
	n.screen.run(false)
	time.Sleep(time.Second * 100)
}

func (n *nes) Run() {
	n.screen.run(true)

	for {
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			if !n.cpu.tick() {
				// so we can run tests
				break
			}
		}

		n.dma.tick()

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3)
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
