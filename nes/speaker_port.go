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

func (s *SpeakerPort) Init() {
	s.doOnce.Do(func() {
		chk(portaudio.Initialize())
		h, err := portaudio.DefaultHostApi()
		chk(err)
		p := portaudio.HighLatencyParameters(nil, h.DefaultOutputDevice)
		p.Output.Channels = 1
		s.stream, err = portaudio.OpenStream(p, s.processAudio)
		chk(err)
		s.sampleRate = int(p.SampleRate)
		s.sampleChan = make(chan float64, s.sampleRate/10)
		chk(s.stream.Start())
	})
}
func (s *SpeakerPort) Reset() {
	s.stream.Stop()
	chk(s.stream.Start())
}
func (s *SpeakerPort) Stop() {
	if s.stream != nil {
		s.stream.Close()
	}
}

func (s *SpeakerPort) Sample(sample float64) bool {
	select {
	case s.sampleChan <- sample:
		return true
	default:
		return false
	}
}
func (s *SpeakerPort) SampleRate() int {
	return s.sampleRate
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
