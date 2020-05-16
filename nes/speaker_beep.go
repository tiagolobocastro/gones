package gones

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"sync"
	"time"
)

type SpeakerBeep struct {
	sampleChan chan float64
	sampleRate beep.SampleRate

	doOnce sync.Once
}

func (s *SpeakerBeep) Init() {
	s.doOnce.Do(func() {
		s.sampleRate = beep.SampleRate(44100)
		s.sampleChan = make(chan float64, s.sampleRate.N(time.Second/10))

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
	select {
	case s.sampleChan <- sample:
		return true
	default:
		return false
	}
}
func (s *SpeakerBeep) SampleRate() int {
	return int(s.sampleRate)
}

func (s *SpeakerBeep) stream() beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		ln := len(samples)
		for i := 0; i < ln; i++ {
			sample := <-s.sampleChan
			samples[i][0] = sample
			samples[i][1] = sample
		}
		return ln, true
	})
}
