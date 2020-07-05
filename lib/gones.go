package lib

import "github.com/tiagolobocastro/gones/lib/nesInternal"

type GoNes interface {
	// Runs the emulator (blocking)
	Run()
	// Requests to...
	Stop()
	Reset()
	// Save/Load the full state of the emulator
	// (excluding some settings like audio library and logging verbosity)
	Save()
	Load()
}

func CartPath(path string) func(n *nesInternal.GoNes) error {
	return nesInternal.CartPath(path)
}

func Verbose(verbose bool) func(n *nesInternal.GoNes) error {
	return nesInternal.Verbose(verbose)
}

func FreeRun(freeRun bool) func(n *nesInternal.GoNes) error {
	return nesInternal.FreeRun(freeRun)
}

func AudioLibrary(name string) func(n *nesInternal.GoNes) error {
	return nesInternal.AudioLibrary(name)
}

func AudioLogging(log bool) func(n *nesInternal.GoNes) error {
	return nesInternal.AudioLogging(log)
}

func SpriteLimit(limit bool) func(n *nesInternal.GoNes) error {
	return nesInternal.SpriteLimit(limit)
}

// Example usage:
// 	nes := gones.NewNES(
//		gones.CartPath("rom.nes"),
//		gones.Verbose(false),
//		gones.AudioLibrary("portaudio"),
//	)
func NewNES(options ...func(n *nesInternal.GoNes) error) GoNes {
	nes := nesInternal.NewNesInternal()

	if err := nes.SetOptions(options...); err != nil {
		panic(err)
	}

	nes.Init()
	return nes.Nes()
}
