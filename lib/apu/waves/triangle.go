package waves

import (
	"github.com/tiagolobocastro/gones/lib/common"
)

type Triangle struct {
	volume uint8

	sequencer Sequencer
	duration  DurationCounter
	linearCnt LinearCounter

	clock   uint64
	period  uint16
	enabled bool
}

func (t *Triangle) Serialise(s common.Serialiser) error {
	return s.Serialise(
		&t.sequencer, &t.duration, &t.linearCnt,
		t.volume, t.clock, t.period, t.enabled,
	)
}
func (t *Triangle) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&t.sequencer, &t.duration, &t.linearCnt,
		&t.volume, &t.clock, &t.period, &t.enabled,
	)
}

func (t *Triangle) Write8(addr uint16, val uint8) {
	switch addr {
	// Length counter halt / linear counter control (C), linear counter load (R)
	case 0x4008:
		t.duration.set((val & 0x80) != 0)
		t.linearCnt.setup((val&0x80) != 0, val&0x7F)

		// Unused
	case 0x4009:

		// Timer low (T)
	case 0x400A:
		t.sequencer.resetLow(val)

	// Length counter load (L), timer high (T)
	case 0x400B:
		// The sequencer is immediately restarted at the first value of the
		// current sequence.
		t.sequencer.resetHigh(val & 0x7)
		t.duration.reload(val >> 3)
		t.linearCnt.start()
	}
}

func (t *Triangle) dutyTable() [][]uint8 {
	//return [][]uint8{
	//	{15, 14, 13, 12, 11, 10, 9, 8,
	//		7, 6, 5, 4, 3, 2, 1, 0,
	//		0, 1, 2, 3, 4, 5, 6, 7, 8,
	//		9, 10, 11, 12, 13, 14, 15},
	//}
	// offset the triangle to get rid of the initial pop as we don't have any low pass filters yet
	// doesn't seem to cause any other noticeable difference
	return [][]uint8{
		{0, 1, 2, 3, 4, 5, 6, 7, 8,
			9, 10, 11, 12, 13, 14, 15,
			15, 14, 13, 12, 11, 10, 9, 8,
			7, 6, 5, 4, 3, 2, 1, 0},
	}
}

func (t *Triangle) setPeriod(period uint16) {
	t.period = period
}
func (t *Triangle) getPeriod() uint16 {
	return t.period
}

func (t *Triangle) Init() {
	t.clock = 0
	t.duration.reset()
	t.sequencer.init(t.dutyTable(), t)
	t.linearCnt.reset()
	t.enabled = false
}
func (t *Triangle) Tick() {
	t.clock++
	if !t.linearCnt.mute() && !t.duration.mute() {
		t.sequencer.tick()
	}
}

func (t *Triangle) QuarterFrameTick() {
	t.linearCnt.tick()
}
func (t *Triangle) HalfFrameTick() {
	t.duration.tick()
}

func (t *Triangle) Sample() float64 {
	output := t.sequencer.value()
	return float64(output)
}
func (t *Triangle) Enabled() bool {
	return !t.duration.mute()
}
func (t *Triangle) Enable(yes bool) {
	t.enabled = yes
	if !yes {
		t.duration.counter = 0
	}
}
