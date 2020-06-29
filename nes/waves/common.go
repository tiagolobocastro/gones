package waves

import (
	"github.com/tiagolobocastro/gones/nes/common"
)

const (
	channelPulse1 = iota
	channelPulse2
	channelTriangle
	channelNoise
	channelDMC
	channelAll1
	channelAll2
)

type timerPeriodInterface interface {
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
		{10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14},
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

func (d *DurationCounter) Serialise(s common.Serialiser) error {
	return s.Serialise(d.counter, d.halt)
}
func (d *DurationCounter) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&d.counter, &d.halt)
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

type Timer struct {
	clock uint

	timer  uint16 // 12bit timer, max val is 4068
	reload uint16
}

func (t *Timer) Serialise(s common.Serialiser) error {
	return s.Serialise(t.clock, t.timer, t.reload)
}
func (t *Timer) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&t.clock, &t.timer, &t.reload)
}

func (t *Timer) reset() {
	t.clock = 0
	t.timer = 0
	t.reload = 0
}
func (t *Timer) set(reload uint16) {
	t.reload = reload
}
func (t *Timer) tick() bool {
	t.clock++

	if t.timer > 0 {
		t.timer--
		return false
	} else {
		t.timer = t.reload
		return true
	}
}
func (t *Timer) value() uint8 {
	return 0
}

type Sequencer struct {
	clock uint

	timer uint16 // 11bit timer

	table  [][]uint8
	width  uint8
	row    uint8
	column uint8

	period timerPeriodInterface
}

func (s *Sequencer) Serialise(sr common.Serialiser) error {
	return sr.Serialise(s.clock, s.timer, s.table, s.width, s.row, s.column)
}
func (s *Sequencer) DeSerialise(sr common.Serialiser) error {
	return sr.DeSerialise(&s.clock, &s.timer, &s.table, &s.width, &s.row, &s.column)
}

func (s *Sequencer) init(table [][]uint8, period timerPeriodInterface) {
	s.table = table
	s.width = uint8(len(table[0]))
	s.column = 0
	s.row = 0
	s.period = period

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
	reload := (s.period.getPeriod() & 0x700) | uint16(value)
	s.period.setPeriod(reload)
}
func (s *Sequencer) resetHigh(value uint8) {
	reload := (s.period.getPeriod() & 0xFF) | (uint16(value) << 8)
	s.period.setPeriod(reload)
}

func (s *Sequencer) tick() {
	s.clock++

	if s.timer > 0 {
		s.timer--
	} else {
		s.timer = s.period.getPeriod()
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

func (e *Envelope) Serialise(s common.Serialiser) error {
	return s.Serialise(e.start, e.loop, e.divider, e.reload, e.decay)
}
func (e *Envelope) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&e.start, &e.loop, &e.divider, &e.reload, &e.decay)
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
// a period channel's period up or down.
// Each sweep unit contains the following: divider, reload flag.
type Sweep struct {
	reload        bool
	enabled       bool
	negate        bool
	shift         uint8
	divider       uint8
	dividerReload uint8

	pulse timerPeriodInterface
}

func (s *Sweep) Serialise(sr common.Serialiser) error {
	return sr.Serialise(s.reload, s.enabled, s.negate, s.shift, s.divider, s.dividerReload)
}
func (s *Sweep) DeSerialise(sr common.Serialiser) error {
	return sr.DeSerialise(&s.reload, &s.enabled, &s.negate, &s.shift, &s.divider, &s.dividerReload)
}

func (s *Sweep) init(pulse timerPeriodInterface) {
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

	// The two period channels have their adders' carry inputs wired differently
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

type LinearCounter struct {
	counterReload uint8
	counter       uint8

	reload  bool
	control bool
}

func (l *LinearCounter) Serialise(s common.Serialiser) error {
	return s.Serialise(l.counterReload, l.counter, l.reload, l.control)
}
func (l *LinearCounter) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&l.counterReload, &l.counter, &l.reload, &l.control)
}

func (l *LinearCounter) reset() {
	l.counter = 0
	l.counterReload = 0
	l.reload = false
	l.control = false
}

func (l *LinearCounter) setup(control bool, reload uint8) {
	l.control = control
	l.counterReload = reload
}

func (l *LinearCounter) start() {
	l.reload = true
}

// When the frame counter generates a linear counter clock, the following actions occur in order:
//
// If the linear counter reload flag is set, the linear counter is reloaded with the counter reload value, otherwise if
// the linear counter is non-zero, it is decremented.
// If the control flag is clear, the linear counter reload flag is cleared.
func (l *LinearCounter) tick() {
	if l.reload {
		l.counter = l.counterReload
	} else if l.counter > 0 {
		l.counter--
	}
	if !l.control {
		l.reload = false
	}
}

func (l *LinearCounter) mute() bool {
	return l.counter == 0
}
