package gones

import (
	"github.com/gordonklaus/portaudio"
	"sync"
)

type SpeakerPort struct {
	sampleChan chan float64
	sampleRate int

	doOnce sync.Once
	stream *portaudio.Stream
}

func (s *SpeakerPort) Init() chan float64 {
	s.doOnce.Do(func() {
		chk(portaudio.Initialize())
		h, err := portaudio.DefaultHostApi()
		chk(err)
		p := portaudio.HighLatencyParameters(nil, h.DefaultOutputDevice)
		p.Output.Channels = 1
		s.stream, err = portaudio.OpenStream(p, s.processAudio)
		chk(err)
		s.sampleRate = int(p.SampleRate)
		s.sampleChan = make(chan float64, s.sampleRate)
		chk(s.stream.Start())
	})
	return s.sampleChan
}

func (s *SpeakerPort) processAudio(out []float32) {
	sample := float32(0.0)
	for i := range out {
		select {
		case apuSample := <-s.sampleChan:
			sample = float32(apuSample)
		default:
		}
		out[i] = sample
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func (s *SpeakerPort) SampleRate() int {
	return s.sampleRate
}
