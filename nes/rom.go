package gones

type rom struct {
	//	busInt
	rom []byte

	writable bool
}

func (r *rom) read8(addr uint16) uint8 {
	return r.rom[addr]
}
func (r *rom) read8w(addr uint32) uint8 {
	return r.rom[addr]
}

// little endian
func (r *rom) read16(addr uint16) uint16 {
	return uint16(r.read8(addr)) | uint16(r.read8(addr+1))<<8
}
func (r *rom) write8(addr uint16, val uint8) {
	r.write8w(addr, val)
}
func (r *rom) write8w(addr uint16, val uint8) {
	if r.writable {
		r.rom[addr] = val
	} else {
		panic("rom is not writable")
	}
}
func (r *rom) write16(addr uint16, val uint16) {
	if r.writable {
		r.write8(addr, uint8(val&0xFF))
		r.write8(addr+1, uint8((val&0xFF00)>>8))
	} else {
		panic("rom is not writable")
	}
}

func (r *rom) size() int {
	return len(r.rom)
}

func (r *rom) init(size int, writable bool) {
	r.rom = make([]byte, size, size)
	r.writable = writable
}
