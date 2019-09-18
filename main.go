package main

import "github.com/tiagolobocastro/gones/nes"

func main() {
	nes := gones.NewNES(true, "C:\\Users\\Tiago\\workspace\\nes-test-roms\\cpu_dummy_reads\\source\\hello_nes\\hello.nes")
	nes.Run()
}
