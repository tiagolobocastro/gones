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

func (s *SpeakerBeep) init() {
	s.doOnce.Do(func() {
		s.sampleRate = beep.SampleRate(44744)
		s.sampleChan = make(chan float64, s.sampleRate.N(time.Second/10))

		speaker.Init(s.sampleRate, s.sampleRate.N(time.Second/10))
		speaker.Play(s.Stream())

		// sound issue with callback lagging behind!
		// looks like we're sampling at slightly above the 44k100
		// which might explain this
		// this dirty hack seems to improve things a bit...
		//s.sampleRate = beep.sampleRate(44100)
	})
}

func (s *SpeakerBeep) Stream() beep.Streamer {
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

func (s *SpeakerBeep) SampleRate() beep.SampleRate {
	return s.sampleRate
}
