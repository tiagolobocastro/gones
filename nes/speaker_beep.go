package gones

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"sync"
	"time"
)

type Speaker struct {
	sampleChan chan float64
	sampleRate beep.SampleRate

	doOnce sync.Once
}

func (s *Speaker) init() {
	s.doOnce.Do(func() {
		s.sampleRate = beep.SampleRate(44100)
		s.sampleChan = make(chan float64, s.sampleRate.N(time.Second/10))

		speaker.Init(s.sampleRate, s.sampleRate.N(time.Second/10))
		speaker.Play(s.Stream())
	})
}

func (s *Speaker) Stream() beep.Streamer {
	//step :=  1 / float64(samplingRate)
	//stepi := 0.0
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		ln := len(samples)
		sample := 0.0
		for i := 0; i < ln; i++ {
			sample = <-s.sampleChan
			samples[i][0] = sample
			samples[i][1] = sample
			//fmt.Printf("%d\n", uint(sample))
		}
		return ln, true
		//nSamples := len(samples)
		//for i := 0; i < nSamples; i++ {
		//
		//	if stepi > (0.5 / 220.0) {
		//
		//		samples[i][0] = 0.2
		//		samples[i][1] = 0.2
		//	} else {
		//
		//		samples[i][0] = 0.0
		//		samples[i][1] = 0.0
		//	}
		//	stepi += step
		//
		//	if stepi > (1.0 / 220.0) {
		//		stepi = 0.0
		//	}
		//}
		//return nSamples, true
	})
}
