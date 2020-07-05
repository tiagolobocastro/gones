package nesInternal

import (
	"fmt"
	"github.com/tiagolobocastro/gones/lib/speakers"
)

func (g *GoNes) SetCart(path string) error {
	g.nes.cartPath = path
	return nil
}
func (g *GoNes) SetVerbose(verbose bool) error {
	g.nes.verbose = verbose
	return nil
}
func (g *GoNes) SetFreeRun(freeRun bool) error {
	g.nes.freeRun = freeRun
	return nil
}
func (g *GoNes) SetAudioLibrary(name speakers.AudioLib) error {
	g.nes.audioLib = name
	return nil
}
func (g *GoNes) SetAudioLogging(log bool) error {
	g.nes.audioLog = log
	return nil
}
func (g *GoNes) SetSpriteLimit(limit bool) error {
	g.nes.spriteLimit = limit
	return nil
}

func (g *GoNes) SetOptions(options ...func(*GoNes) error) error {
	for i, option := range options {
		if err := option(g); err != nil {
			return fmt.Errorf("failed to set option index %d, err=%v", i, err)
		}
	}
	return nil
}

func CartPath(path string) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetCart(path)
	}
}

func Verbose(verbose bool) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetVerbose(verbose)
	}
}

func FreeRun(freeRun bool) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetFreeRun(freeRun)
	}
}

func AudioLibrary(name string) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetAudioLibrary(speakers.AudioLib(name))
	}
}

func AudioLogging(log bool) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetAudioLogging(log)
	}
}

func SpriteLimit(limit bool) func(n *GoNes) error {
	return func(n *GoNes) error {
		return n.SetSpriteLimit(limit)
	}
}
