package gones

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

func NewNES(options ...func(*nes) error) *nes {
	nes := &nes{}
	nes.audioLib = Nil

	if err := nes.setOptions(options...); err != nil {
		panic(err)
	}

	nes.init()
	return nes
}

func (n *nes) Stop() {
	n.apu.Stop()
}

func (n *nes) Reset() {
	n.resetRq = true
}

func (n *nes) Run() {
	n.screen.run()
	if n.freeRun == true {
		n.runFree()
	} else {
		tmr := time.Tick(time.Second / 240)
		for !n.ApuBufferReady() {
			// pre-fill enough sound samples
			n.Step((time.Second / 240).Seconds())
			<-tmr
		}
		go func() {
			tmr := time.Tick(time.Second / 240)
			n.apu.Play()
			for {
				n.Step((time.Second / 240).Seconds())
				<-tmr
			}
		}()
		for {
			time.Sleep(time.Second * 100)
		}
	}
}

func (n *nes) init() {
	n.bus.Init()

	if err := n.cart.Init(n.cartPath); err != nil {
		log.Panicf("Failed to initialise the cartridge, err=%v", err)
	}

	n.ram.Init(0x800)

	n.ctrl.init()
	n.screen.init(n)

	n.cpu.Init(n.bus.GetBusInt(MapCPUId), n.verbose)
	n.ppu.Init(n.bus.GetBusInt(MapPPUId), &n.cpu, n.verbose, &n.screen.framebuffer, n.spriteLimit)
	n.dma.init(n.bus.GetBusInt(MapDMAId))
	n.apu.init(n.bus.GetBusInt(MapAPUId), &n.cpu, n.verbose, n.audioLog, n.audioLib)

	n.bus.Connect(MapCPUId, &cpuMapper{n})
	n.bus.Connect(MapPPUId, &ppuMapper{n})
	n.bus.Connect(MapDMAId, &dmaMapper{n})
	n.bus.Connect(MapAPUId, &apuMapper{n})

	n.cpu.Reset()
}

func (n *nes) stats() {
	n.cpu.Stats()
}

func (n *nes) Step(seconds float64) {
	cyclesPerSecond := float64(NesBaseFrequency)
	cyclesPerSecond *= seconds
	runCycles := int(cyclesPerSecond)

	for runCycles > 0 {

		ticks := 1
		if !n.dma.active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.Tick()
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.Ticks(3 * ticks)
		n.dma.ticks(ticks)

		// since we are more sensitive to sound
		// so we might have to replace the cpu as the "tick master"
		n.apu.ticks(ticks)

		runCycles -= ticks
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
			ticks = n.cpu.Tick()
		}

		if ticks == 0 {
			return
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.Ticks(3 * ticks)
		n.dma.ticks(ticks)
	}
}

func (n *nes) ApuBufferReady() bool {
	return n.apu.audioBufferReady()
}

func (n *nes) runFree() {
	for !n.ApuBufferReady() {
		// pre-fill enough sound samples
		n.Step((time.Second / 240).Seconds())
	}
	n.apu.Play()

	for {
		for {
			n.Step(time.Second.Seconds())
		}
	}
}

func (n *nes) reset() {
	// probably need to stall them first
	n.ppu.Reset()
	n.dma.reset()
	n.cpu.Reset()
	n.apu.reset()

	n.resetRq = false
}

// loads hex dumps from: https://skilldrick.github.io/easy6502/, eg:
// `0600: a9 01 85 02 a9 cc 8d 00 01 a9 01 a a1 00 00 00
//  0610: a9 05 a 8e 00 02 a9 05 8d 01 02 a9 08 8d 02 02`

func (n *nes) loadEasyCode(code string) {

	for i, line := range strings.Split(strings.TrimSuffix(code, "\n"), "\n") {
		addr := 0
		var bt [16]int
		ns, err := fmt.Sscanf(line, "%X: %X %X %X %X %X %X %X %X %X %X %X %X %X %X %X %X ",
			&addr, &bt[0], &bt[1], &bt[2], &bt[3], &bt[4], &bt[5], &bt[6], &bt[7],
			&bt[8], &bt[9], &bt[10], &bt[11], &bt[12], &bt[13], &bt[14], &bt[15])
		if err != nil && err != io.EOF {
			log.Printf("Error when scanning easyCode line, ns: %X, error: %v\n", ns, err)
		}

		if i == 0 {
			// assumes first line is where the program starts
			n.cart.WriteRom16(0xFFFC, uint16(addr))
		}

		for i, b := range bt {
			n.cpu.Write8(uint16(addr+i), uint8(b))
		}
	}
}
