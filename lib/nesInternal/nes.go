package nesInternal

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/tiagolobocastro/gones/lib/apu"
	"github.com/tiagolobocastro/gones/lib/common"
	"github.com/tiagolobocastro/gones/lib/cpu"
	"github.com/tiagolobocastro/gones/lib/mappers"
	"github.com/tiagolobocastro/gones/lib/ppu"
	"github.com/tiagolobocastro/gones/lib/speakers"
)

type GoNes struct {
	nes *nes
}

func NewNesInternal() *GoNes {
	return &GoNes{&nes{audioLib: speakers.Nil}}
}
func (g *GoNes) Init() {
	g.nes.init()
}
func (g *GoNes) Nes() *nes {
	return g.nes
}

func (n *nes) Stop() {
	n.cart.Stop()
	n.apu.Stop()
}

func (n *nes) Request(request NesOpRequest) {
	n.opRequests |= 1 << request
}

func (n *nes) Reset() {
	n.opRequests |= 1 << ResetRequest
}

func (n *nes) Save() {
	n.opRequests |= 1 << SaveRequest
}

func (n *nes) Load() {
	n.opRequests |= 1 << LoadRequest
}

func (n *nes) Poke(controllerId uint8, button uint8, pressed bool) {
	n.ctrl.Poke(controllerId, button, pressed)
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

func (n *nes) PPU() *ppu.Ppu {
	return &n.ppu
}
func (n *nes) CPU() *cpu.Cpu {
	return &n.cpu
}

func (n *nes) init() {
	n.bus.Init()

	if err := n.cart.Init(n.cartPath, n); err != nil {
		log.Panicf("Failed to initialise the cartridge, err=%v", err)
	}

	n.ram.Init(0x800)

	n.ctrl.Init()
	n.screen.init(n)

	n.cpu.Init(n.bus.GetBusInt(MapCPUId), n.verbose)
	n.ppu.Init(n.bus.GetBusInt(MapPPUId), &n.cpu, n.verbose, &n.screen.framebuffer, n.spriteLimit)
	n.dma.Init(n.bus.GetBusInt(MapDMAId))
	n.apu.Init(n.bus.GetBusInt(MapAPUId), &n.cpu, n.verbose, n.audioLog, n.audioLib)

	n.bus.Connect(MapCPUId, &cpuMapper{n})
	n.bus.Connect(MapPPUId, &ppuMapper{n})
	n.bus.Connect(MapDMAId, &dmaMapper{n})
	n.bus.Connect(MapAPUId, &apuMapper{n})

	n.cpu.Reset()
}

func (n *nes) reset() {
	n.ppu.Reset()
	n.dma.Reset()
	n.cpu.Reset()
	n.apu.Reset()
	n.ctrl.Reset()
	n.cart.Reset()

	n.opRequests &= ^(1 << ResetRequest)
}

func (n *nes) save() {
	err := n.Serialise(common.NewSerialiser(n.cart.GetStateSaveFile()))
	if err != nil {
		log.Printf("Failed to Save State: %v", err)
	}
	n.opRequests &= ^(1 << SaveRequest)
}

func (n *nes) load() {
	// we need to reset the nest because otherwise the gob encoder
	// does a gob out: https://github.com/golang/go/issues/21929
	n.reset()
	if err := n.DeSerialise(common.NewSerialiser(n.cart.GetStateSaveFile())); err != nil {
		log.Printf("Failed to Load State: %v", err)
	}
	n.opRequests &= ^(1 << LoadRequest)
}

func (n *nes) Serialise(s common.Serialiser) error {
	return s.Serialise(
		&n.cpu, &n.ram, &n.apu, &n.dma, &n.ppu, &n.cart, &n.screen, &n.ctrl,
		n.opRequests, n.freeRun, n.audioLib, n.audioLog, n.spriteLimit,
	)
}

func (n *nes) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&n.cpu, &n.ram, &n.apu, &n.dma, &n.ppu, &n.cart, &n.screen, &n.ctrl,
		&n.opRequests, &n.freeRun, &n.audioLib, &n.audioLog, &n.spriteLimit,
	)
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
		if !n.dma.Active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.Tick()
		}

		// 3 ppu ticks per 1 cpu
		for i := 0; i < 3*ticks; i++ {
			n.ppu.Ticks(1)
			n.cart.Ticks(1)
		}

		n.dma.Ticks(ticks)

		// since we are more sensitive to sound
		// so we might have to replace the cpu as the "tick master"
		n.apu.Ticks(ticks)

		runCycles -= ticks
	}

	n.processOpRequest()
}

func (n *nes) processOpRequest() {
	switch {
	case n.opRequests&(1<<ResetRequest) != 0:
		n.reset()
	case n.opRequests&(1<<SaveRequest) != 0:
		n.save()
	case n.opRequests&(1<<LoadRequest) != 0:
		n.load()
	}
}

func (n *nes) Test() {
	for {
		ticks := 1
		if !n.dma.Active() {
			// cpu stalled whilst dma is active
			ticks = n.cpu.Tick()
		}

		if ticks == 0 {
			return
		}

		// 3 ppu ticks per 1 cpu
		n.ppu.Ticks(3 * ticks)
		n.dma.Ticks(ticks)
	}
}

func (n *nes) ApuBufferReady() bool {
	return n.apu.AudioBufferReady()
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

const (
	frameXWidth  = 256
	frameYHeight = 240

	screenFrameRatio = 3
	screenXWidth     = frameXWidth * screenFrameRatio
	screenYHeight    = frameYHeight * screenFrameRatio
)

type NesOpRequest int

const (
	ResetRequest NesOpRequest = iota
	SaveRequest
	LoadRequest
)

type nes struct {
	bus common.Bus

	cpu  cpu.Cpu
	ram  common.Ram
	cart mappers.Cartridge
	ppu  ppu.Ppu
	dma  common.Dma
	apu  apu.Apu
	ctrl common.Controllers

	screen screen

	opRequests NesOpRequest

	// Options
	verbose     bool
	cartPath    string
	freeRun     bool
	audioLib    speakers.AudioLib
	audioLog    bool
	spriteLimit bool
}

const (
	MapCPUId = iota
	MapPPUId
	MapDMAId
	MapAPUId
)

const NesBaseFrequency = 1789773 // NTSC
//const NesBaseFrequency = 1662607 // PAL
