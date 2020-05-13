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

type pulseInterface interface {
	setPeriod(uint16)
	getPeriod() uint16
}

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
func (d *DurationCounter) mute() bool {
	return d.counter == 0
}

type Sequencer struct {
	clock uint

	timer uint16 // 11bit timer

	table  [][]uint8
	width  uint8
	row    uint8
	column uint8

	pulse pulseInterface
}

func (s *Sequencer) init(table [][]uint8, pulse pulseInterface) {
	s.table = table
	s.width = uint8(len(table[0]))
	s.column = 0
	s.row = 0
	s.pulse = pulse

	s.reset()
}
func (s *Sequencer) reset() {
	s.clock = 0
	s.timer = 0
}

func (s *Sequencer) selectRow(row uint8) {
	s.row = row
}
func (s *Sequencer) resetLow(value uint8) {
	reload := (s.pulse.getPeriod() & 0x700) | uint16(value)
	s.pulse.setPeriod(reload)
}
func (s *Sequencer) resetHigh(value uint8) {
	reload := (s.pulse.getPeriod() & 0xFF) | (uint16(value) << 8)
	s.pulse.setPeriod(reload)
	s.column = 0
}

func (s *Sequencer) tick() {
	s.clock++

	if s.timer > 0 {
		s.timer--
	} else {
		s.timer = s.pulse.getPeriod()
		s.column = (s.column + 1) % s.width
	}
}

func (s *Sequencer) value() uint8 {
	return s.table[s.row][s.column]
}

// Each volume envelope unit contains the following:
// start flag, divider, and decay level counter.
type Envelope struct {
	start   bool
	loop    bool
	divider uint8
	reload  uint8
	decay   uint8
}

func (e *Envelope) reset() {
	e.start = false
	e.loop = false
	e.divider = 0
	e.reload = 0
	e.decay = 0
}

func (e *Envelope) tick() {
	if !e.start {
		if e.divider == 0 {
			e.divider = e.reload
			if e.decay > 0 {
				e.decay--
			} else if e.loop {
				e.decay = 15
			}
		} else {
			e.divider--
		}
	} else {
		e.start = false
		e.decay = 15
		e.divider = e.reload
	}
}

// An NES APU sweep unit can be made to periodically adjust
// a pulse channel's period up or down.
// Each sweep unit contains the following: divider, reload flag.
type Sweep struct {
	reload        bool
	enabled       bool
	negate        bool
	shift         uint8
	divider       uint8
	dividerReload uint8

	pulse pulseInterface
}

func (s *Sweep) init(pulse pulseInterface) {
	s.pulse = pulse
}

func (s *Sweep) tick() {

	if s.divider == 0 && s.enabled && !s.mute() {
		// adjust period
		s.updatePeriod()
	}

	if s.divider == 0 || s.reload {
		s.reload = false
		s.divider = s.dividerReload
	} else {
		s.divider--
	}
}

// a target period overflow from the sweep unit's adder can silence a channel even
// when the enabled flag is clear and even when the sweep divider is not outputting
// a clock signal. Thus to fully disable the sweep unit, a program must turn off
// enable and turn on negate, such as by writing $08.
// This ensures that the target period is not greater than the current period and
// therefore not greater than $7FF.
func (s *Sweep) mute() bool {
	return s.targetPeriod() > 0x7FF ||
		// This avoids sending harmonics in the hundreds of kHz through the audio path.
		// Muting based on a too-small current period cannot be overridden.
		s.targetPeriod() < 8
}

func (s *Sweep) updatePeriod() {
	period := s.targetPeriod()
	s.pulse.setPeriod(period)
}

func (s *Sweep) targetPeriod() uint16 {
	rawPeriod := s.pulse.getPeriod()
	change := rawPeriod >> s.shift

	// The two pulse channels have their adders' carry inputs wired differently
	// which produces different results when each channel's change amount is made negative:
	//
	// Pulse 1 adds the ones' complement (−c − 1).
	// -> Making 20 negative produces a change amount of −21.
	// Pulse 2 adds the two's complement (−c).
	// -> Making 20 negative produces a change amount of −20.
	if s.negate {
		return rawPeriod - change
	} else {
		return rawPeriod + change
	}

	// Whenever the current period changes for any reason, whether by $400x writes or by sweep,
	// the target period also changes.
}
