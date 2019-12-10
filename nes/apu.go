package gones

const (
	channelPulse1 = iota
	channelPulse2
	channelTriangle
	channelNoise
	channelDMC
	channelAll1
	channelAll2
)

type pulse struct {
}

func (a *apu) init() {

}

func (a *apu) read8(addr uint16) uint8 {
	return 0
}

func (a *apu) write8(addr uint16, val uint8) {
	switch {
	case addr > 1:
	}
}
