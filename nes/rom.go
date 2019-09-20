package gones

type rom struct {
	//	busInt
	rom []byte
}

func (r *rom) read8(addr uint16) uint8 {
	return r.rom[addr]
}

// little endian
func (r *rom) read16(addr uint16) uint16 {
	return uint16(r.read8(addr)) | uint16(r.read8(addr+1))<<8
}

func (r *rom) write8(uint16, uint8) {
	panic("rom is not writable")
}
func (r *rom) write16(uint16, uint16) {
	panic("rom is not writable")
}

func (r *rom) size() uint16 {
	return uint16(len(r.rom))
}

func (r *rom) init(size int) {
	r.rom = make([]byte, size, size)
}
