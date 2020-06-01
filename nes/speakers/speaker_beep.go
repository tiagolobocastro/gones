package speakers

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"time"
)

type SpeakerBeep struct {
	sampleRate  beep.SampleRate
	speakerSize int
	buffer      *CircularBuffer
}

func (s *SpeakerBeep) Init() {
	s.sampleRate = beep.SampleRate(44100)
	s.buffer = NewCircularBuffer(s.sampleRate.N(time.Second) / 10)
	s.speakerSize = s.sampleRate.N(time.Second / 100)

	speaker.Init(s.sampleRate, s.speakerSize)
}

func (s *SpeakerBeep) Play() {
	speaker.Play(s.stream())
}
func (s *SpeakerBeep) Reset() {}
func (s *SpeakerBeep) Stop() {
	if s.sampleRate != 0 {
		speaker.Close()
	}
}
func (s *SpeakerBeep) BufferReady() bool {
	return s.buffer.available() > int(float64(s.speakerSize)*1.5)
}
func (s *SpeakerBeep) Sample(sample float64) bool {
	if s.buffer.Write(sample, false) != nil {
		// this will cause issues but might be better than
		// increasing the latency between the apu and the speaker
		_, _ = s.buffer.Read()
		return false
	}
	return true
}

func (s *SpeakerBeep) SampleRate() int {
	return int(s.sampleRate)
}
func (s *SpeakerBeep) stream() beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		read := s.buffer.ReadInto2(samples)
		return read, true
	})
}
