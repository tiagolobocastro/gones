package gones

type SpeakerNil struct {
	sampleChan chan float64
	sampleRate int
}

func (s *SpeakerNil) Init() {
	s.sampleRate = 44100
	s.sampleChan = make(chan float64, s.sampleRate/10)
}
func (s *SpeakerNil) Reset() {}
func (s *SpeakerNil) Play()  {}
func (s *SpeakerNil) Stop()  {}

func (s *SpeakerNil) Sample(float64) bool {
	return true
}
func (s *SpeakerNil) SampleRate() int {
	return s.sampleRate
}
func (s *SpeakerNil) BufferReady() bool {
	return true
}
