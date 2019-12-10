package main

import (
	"flag"
	"fmt"
	"os"

	gones "github.com/tiagolobocastro/gones/nes"
)

const defaultINesPath = `C:\Users\Tiago\Dropbox\nes\Donkey Kong Jr. (JU).nes`

// const defaultINesPath = `C:\Users\Tiago\Dropbox\nes\Donkey Kong Jr. (JU).nes`
// const defaultINesPath = `C:\Users\Tiago\Dropbox\nes\Super Mario Bros. (World).nes`
// const defaultINesPath = `C:\Users\Tiago\Dropbox\nes\Donkey Kong (World) (Rev A).nes`

func validINesPath(romPath string) error {

	stat, err := os.Stat(romPath)
	if err != nil {
		return fmt.Errorf("iNes Rom file path (\"%v\") does not exist or is not valid", romPath)
	} else if stat.IsDir() {
		return fmt.Errorf("iNes Rom file path (\"%v\") points to a directory", romPath)
	}
	return nil
}

func main() {
	romPath := flag.String("rom", defaultINesPath, "path to the iNes Rom file to run")
	flag.Parse()

	if err := validINesPath(*romPath); err != nil {
		fmt.Printf("Failed to start GoNes, err=%v\n", err)
		return
	}

	fmt.Printf("Starting GoNes with iNes Rom file: %s\n", *romPath)
	nes := gones.NewNES(gones.CartPath(*romPath), gones.Verbose(false), gones.FreeRun(false))
	nes.Run()
}
