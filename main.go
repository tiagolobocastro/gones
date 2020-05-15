package main

import (
	"flag"
	"fmt"
	"os"

	gones "github.com/tiagolobocastro/gones/nes"
)

const defaultAudioLibrary = gones.Beep

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
	romPath := flag.String("rom", "", "path to the iNes Rom file to run")
	audioLib := flag.String("speaker", defaultAudioLibrary, "portaudio or beep speaker engine")
	logAudio := flag.Bool("logAudio", false, "log audio sampling average every second")

	flag.Parse()

	if len(os.Args) > 1 && *romPath == "" {
		*romPath = os.Args[1]
	}

	if err := validateINesPath(*romPath); err != nil {
		fmt.Printf("Failed to start GoNes, err=%v\n", err)
		return
	}

	fmt.Printf("Starting GoNes with iNes Rom file: %s\n", *romPath)
	nes := gones.NewNES(
		gones.CartPath(*romPath),
		gones.Verbose(false),
		gones.FreeRun(false),
		gones.AudioLibrary(*audioLib),
		gones.AudioLogging(*logAudio),
	)

	nes.Run()
}
