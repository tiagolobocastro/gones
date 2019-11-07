package main

import gones "github.com/tiagolobocastro/gones/nes"

func main() {
	//nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\cpu_dummy_reads\\source\\hello_nes\\hello.nes")
	// nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\sprites\\sprites.nes")
	nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\sprite_movement\\spritemovement.nes")
	//nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\background2\\background.nes")
	//nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\full_palette\\flowing_palette.nes")
	//nes := gones.NewNES(false, "C:\\Users\\Tiago\\Downloads\\Donkey Kong (World) (Rev A).nes")
	nes.Run2()
}
