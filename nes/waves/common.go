package waves

const (
	channelPulse1 = iota
	channelPulse2
	channelTriangle
	channelNoise
	channelDMC
	channelAll1
	channelAll2
)

type frameCounterInt interface {
	QuarterFrameTick()
	HalfFrameTick()
}

func DurationCounterTable(load uint8) uint8 {
	// https://wiki.nesdev.com/w/index.php/APU_Length_Counter
	//      |  0   1   2   3   4   5   6   7    8   9   A   B   C   D   E   F
	// -----+----------------------------------------------------------------
	// 00-0F  10,254, 20,  2, 40,  4, 80,  6, 160,  8, 60, 10, 14, 12, 26, 14,
	// 10-1F  12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30
	table := [][]uint8{
		{0, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14},
		{12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30},
	}
	tableIndex := (load & 0x10) >> 4
	valueIndex := load & 0xF
	return table[tableIndex][valueIndex]
}

// also called the lenCounter
type DurationCounter struct {
	counter uint8
	halt    bool
}

func (d *DurationCounter) tick() {
	if !d.halt && d.counter > 0 {
		d.counter--
	}
}
func (d *DurationCounter) reset() {
	d.counter = 0
	d.halt = true
}
func (d *DurationCounter) set(halt bool) {
	d.halt = halt
}
func (d *DurationCounter) reload(val uint8) {
	d.counter = DurationCounterTable(val)
}

type Sequencer struct {
	clock uint

	timer  uint16 // 11bit timer
	reload uint16
}

func (s *Sequencer) reset() {
	s.clock = 0
	s.timer = 0
	s.reload = 0
}
func (s *Sequencer) resetLow(value uint8) {
	s.reload = (s.reload & 0x700) | uint16(value)
}
func (s *Sequencer) resetHigh(value uint8) {
	s.reload = (s.reload & 0xFF) | (uint16(value) << 8)
}

func (s *Sequencer) tick() bool {
	s.clock++

	if s.timer > 0 {
		s.timer--
		return false
	} else {
		s.timer = s.reload
		return true
	}
}
