package speakers

type AudioLib string

const (
	Nil       = "nil"
	Beep      = "beep"
	PortAudio = "portaudio"
	Oto       = "oto"
)

type AudioSpeaker interface {
	Init()
	Reset()
	Stop()
	Play()
	Sample(float64) bool
	SampleRate() int
	BufferReady() bool
}

func NewSpeaker(lib AudioLib) AudioSpeaker {
	var speaker AudioSpeaker
	switch lib {
	case Nil:
		speaker = new(SpeakerNil)
	case Beep:
		speaker = new(SpeakerBeep)
	case PortAudio:
		speaker = new(SpeakerPort)
	case Oto:
		speaker = new(SpeakerOto)
	default:
		panic("Unknown speaker type!")
	}
	speaker.Init()
	return speaker
}
