package speakers

import (
	"github.com/gordonklaus/portaudio"
)

type SpeakerPort struct {
	buffer  *CircularBuffer
	playing bool
	stream  *portaudio.Stream
}

func (s *SpeakerPort) Init() {
	chk(portaudio.Initialize())
	h, err := portaudio.DefaultHostApi()
	chk(err)
	p := portaudio.HighLatencyParameters(nil, h.DefaultOutputDevice)
	p.Output.Channels = 1
	s.stream, err = portaudio.OpenStream(p, s.processAudio)
	chk(err)
	s.playing = false
	s.buffer = NewCircularBuffer(int(p.SampleRate))
}
func (s *SpeakerPort) Reset() {
	if s.playing {
		chk(s.stream.Stop())
		chk(s.stream.Start())
	}
}
func (s *SpeakerPort) Play() {
	chk(s.stream.Start())
	s.playing = true
}
func (s *SpeakerPort) Stop() {
	_ = s.stream.Close()
	s.playing = false
}
func (s *SpeakerPort) BufferReady() bool {
	return s.buffer.available() > int(s.stream.Info().SampleRate*0.3)
}

func (s *SpeakerPort) Sample(sample float64) bool {
	return s.buffer.Write(sample, false) == nil
}
func (s *SpeakerPort) SampleRate() int {
	return int(s.stream.Info().SampleRate)
}

func (s *SpeakerPort) processAudio(out []float32) {
	_ = s.buffer.ReadInto(out)
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
