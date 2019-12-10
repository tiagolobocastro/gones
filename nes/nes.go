package gones

import (
	"log"
	"time"
)

const nesBaseFrequency = 1789773

func (n *nes) init() {
	n.bus.init()

	if err := n.cart.init(n.cartPath); err != nil {
		panic(err)
	}

	n.ram.init(0x800)
	n.vRam.init(0x800)

	n.ctrl.init()
	n.screen.init(n)

	n.cpu.init(n.bus.getBusInt(MapCPUId), n.verbose)
	n.ppu.init(n.bus.getBusInt(MapPPUId), n.verbose, &n.cpu, &n.screen.framebuffer)
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
	cyclesPerSecond := float64(nesBaseFrequency)
	cyclesPerSecond *= seconds
	runCycles := int(cyclesPerSecond)

	//frames := n.ppu.frameBuffer.frames
	for runCycles > 0 {

		ticks := 1
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.tick()
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3 * ticks)
		n.dma.ticks(ticks)

		runCycles -= ticks
		// use this to step a whole frame at a time
		// if n.ppu.frameBuffer.frames > frames {
		//	return
		// }
	}

	if n.resetRq {
		n.reset()
	}
}

func (n *nes) Test() {
	for {
		ticks := 1
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.tick()
		}

		if ticks == 0 {
			return
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3 * ticks)
		n.dma.ticks(ticks)
	}
}

func (n *nes) runFree() {
	for {
		ticks := 1
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.tick()
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.ticks(3 * ticks)
		n.dma.ticks(ticks)
	}
}

func (n *nes) Run() {

	n.screen.run(n.freeRun)
	if n.freeRun == true {
		n.runFree()
	} else {
		for {
			time.Sleep(time.Second * 100)
		}
	}
}

func (n *nes) reset() {
	// probably need to stall them first
	n.ppu.reset()
	n.dma.reset()
	n.cpu.reset()

	n.resetRq = false
}

func (n *nes) resetRequest() {
	n.resetRq = true
}

func NewNES(options ...func(*nes) error) *nes {
	nes := &nes{}

	if err := nes.setOptions(options...); err != nil {
		panic(err)
	}

	nes.init()
	return nes
}
