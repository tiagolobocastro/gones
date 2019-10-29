package main

import "github.com/tiagolobocastro/gones/nes"

func main() {

	//nes := gones.NewNES(true, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\cpu_dummy_reads\\source\\hello_nes\\hello.nes")
	// nes := gones.NewNES(true, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\sprites\\sprites.nes")
	// nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\sprite_movement\\spritemovement.nes")
	nes := gones.NewNES(false, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\pong1\\pong1.nes")
	nes.Run()
}
