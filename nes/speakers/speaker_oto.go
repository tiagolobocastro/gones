package speakers

import (
	"github.com/hajimehoshi/oto"
)

type SpeakerOto struct {
	sampleRate  int
	speakerSize int
	buffer      *CircularBuffer

	samples [][2]float64
	buf     []byte
	context *oto.Context
	player  *oto.Player
}

func (s *SpeakerOto) Init() {
	s.sampleRate = 44100
	s.buffer = NewCircularBuffer(s.sampleRate / 10)
	s.speakerSize = s.sampleRate / 100

	s.otoInit()
}

func (s *SpeakerOto) otoInit() {
	numBytes := s.speakerSize * 4
	s.samples = make([][2]float64, s.speakerSize)
	s.buf = make([]byte, numBytes)

	s.context, _ = oto.NewContext(s.sampleRate, 2, 2, numBytes)
}

func (s *SpeakerOto) Play() {
	s.player = s.context.NewPlayer()
}
func (s *SpeakerOto) Reset() {}
func (s *SpeakerOto) Stop() {
	s.player.Close()
	s.context.Close()
	s.player = nil
}
func (s *SpeakerOto) BufferReady() bool {
	return s.buffer.available() > int(float64(s.speakerSize)*1.5)
}
func (s *SpeakerOto) Sample(sample float64) bool {
	if s.buffer.Write(sample, false) != nil {
		// this will cause issues but might be better than
		// increasing the latency between the apu and the speaker
		_, _ = s.buffer.Read()
		return false
	}

	if s.buffer.Available() > 2048 && s.player != nil {
		s.buffer.ReadInto2(s.samples)
		go s.update()
	}

	return true
}

func (s *SpeakerOto) update() {

	for i := range s.samples {
		for c := range s.samples[i] {
			val := s.samples[i][c]
			if val < -1 {
				val = -1
			}
			if val > +1 {
				val = +1
			}
			valInt16 := int16(val * (1<<15 - 1))
			low := byte(valInt16)
			high := byte(valInt16 >> 8)
			s.buf[i*4+c*2+0] = low
			s.buf[i*4+c*2+1] = high
		}
	}

	s.player.Write(s.buf)
}

func (s *SpeakerOto) SampleRate() int {
	return s.sampleRate
}
