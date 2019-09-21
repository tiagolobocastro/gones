package gones

type ram struct {
	//	busInt

	ram []byte
}

func (r *ram) size() uint16 {
	return uint16(len(r.ram))
}

func (r *ram) init(size int) {
	r.ram = make([]byte, size)
}

func (r *ram) read8(addr uint16) uint8 {
	return r.ram[addr]
}
func (r *ram) write8(addr uint16, val uint8) {
	r.ram[addr] = val
}

// little endian
func (r *ram) read16(addr uint16) uint16 {
	return uint16(r.read8(addr)) | uint16(r.read8(addr+1))<<8
}
func (r *ram) write16(addr uint16, val uint16) {
	r.write8(addr, uint8(val&0xFF))
	r.write8(addr+1, uint8((val&0xFF00)>>8))
}
