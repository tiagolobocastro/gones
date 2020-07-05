package main

import (
	"flag"
	"fmt"
	"os"

	gones "github.com/tiagolobocastro/gones/lib"
	"github.com/tiagolobocastro/gones/lib/speakers"
)

const defaultAudioLibrary = speakers.Beep

func validateINesPath(romPath string) error {
	stat, err := os.Stat(romPath)
	if err != nil {
		return fmt.Errorf("iNes Rom file path (\"%v\") does not exist or is not valid", romPath)
	} else if stat.IsDir() {
		return fmt.Errorf("iNes Rom file path (\"%v\") points to a directory", romPath)
	}
	return nil
}

func main() {
	romPath := ""
	positionalArgs := 0
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		romPath = os.Args[1]
		positionalArgs++
	}

	flag.StringVar(&romPath, "rom", romPath, "path to the iNes Rom file to run")
	audioLib := flag.String("audio", defaultAudioLibrary, "beep, portaudio or nil")
	logAudio := flag.Bool("logaudio", false, "log audio sampling average every second (debug only)")
	verbose := flag.Bool("verbose", false, "verbose logs (debug only)")
	freeRun := flag.Bool("freerun", false, "run as fast as possible with double buffered sync (debug only)")
	spriteLimit := flag.Bool("spritelimit", false, "limit number of sprites per scanline to 8 (true to the NES)")
	if err := flag.CommandLine.Parse(os.Args[positionalArgs+1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to parse the commandline parameters, err=%v\n", err)
		return
	}

	if err := validateINesPath(romPath); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Rom image path is not valid? err=%v\n", err)
		return
	}

	fmt.Printf("Starting GoNes with iNes Rom file: %s\n", romPath)
	nes := gones.NewNES(
		gones.CartPath(romPath),
		gones.Verbose(*verbose),
		gones.FreeRun(*freeRun),
		gones.AudioLibrary(*audioLib),
		gones.AudioLogging(*logAudio),
		gones.SpriteLimit(*spriteLimit),
	)

	nes.Run()
}
