package gones

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"sync"
	"time"
)

type SpeakerBeep struct {
	sampleRate beep.SampleRate
	buffer     CircularBuffer

	doOnce sync.Once
}

func (s *SpeakerBeep) Init() {
	s.doOnce.Do(func() {
		s.sampleRate = beep.SampleRate(44100)
		s.buffer = NewCircularBuffer(s.sampleRate.N(time.Second))

		speaker.Init(s.sampleRate, s.sampleRate.N(time.Second/10))
		speaker.Play(s.stream())
	})
}
func (s *SpeakerBeep) Reset() {}
func (s *SpeakerBeep) Stop() {
	if s.sampleRate != 0 {
		speaker.Close()
	}
}

func (s *SpeakerBeep) Sample(sample float64) bool {
	if s.buffer.Write(sample, false) != nil {
		s.buffer.Read()
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
