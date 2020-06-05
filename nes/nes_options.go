package gones

import "fmt"

func (n *nes) setOptions(options ...func(*nes) error) error {
	for i, option := range options {
		if err := option(n); err != nil {
			return fmt.Errorf("failed to set option index %d, err=%v", i, err)
		}
	}
	return nil
}

func (n *nes) setCart(path string) error {
	n.cartPath = path
	return nil
}
func (n *nes) setVerbose(verbose bool) error {
	n.verbose = verbose
	return nil
}
func (n *nes) setFreeRun(freeRun bool) error {
	n.freeRun = freeRun
	return nil
}
func (n *nes) setAudioLibrary(name AudioLib) error {
	n.audioLib = name
	return nil
}
func (n *nes) setAudioLogging(log bool) error {
	n.audioLog = log
	return nil
}
func (n *nes) setSpriteLimit(limit bool) error {
	n.spriteLimit = limit
	return nil
}

func CartPath(path string) func(n *nes) error {
	return func(n *nes) error {
		return n.setCart(path)
	}
}

func Verbose(verbose bool) func(n *nes) error {
	return func(n *nes) error {
		return n.setVerbose(verbose)
	}
}

func FreeRun(freeRun bool) func(n *nes) error {
	return func(n *nes) error {
		return n.setFreeRun(freeRun)
	}
}

func AudioLibrary(name string) func(n *nes) error {
	return func(n *nes) error {
		return n.setAudioLibrary(AudioLib(name))
	}
}

func AudioLogging(log bool) func(n *nes) error {
	return func(n *nes) error {
		return n.setAudioLogging(log)
	}
}

func SpriteLimit(limit bool) func(n *nes) error {
	return func(n *nes) error {
		return n.setSpriteLimit(limit)
	}
}
