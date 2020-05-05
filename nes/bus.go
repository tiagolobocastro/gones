package gones

type busInt interface {
	// Data Operations
	read8(uint16) uint8
	write8(uint16, uint8)
}

type busExtInt interface {
	// Data Operations
	read8(uint16) uint8
	write8(uint16, uint8)
	read16(uint16) uint16
	write16(uint16, uint16)
}

type bus struct {
	//	busInt
	maps []BusMapInt
}

type BusMapInt struct {
	//	busInt
	mapId uint

	busInt
}

func (b *BusMapInt) read8(addr uint16) uint8 {
	return b.busInt.read8(addr)
}
func (b *BusMapInt) read16(addr uint16) uint16 {
	return uint16(b.read8(addr)) | uint16(b.read8(addr+1))<<8
}

func (b *BusMapInt) write8(addr uint16, val uint8) {
	b.busInt.write8(addr, val)
}
func (b *BusMapInt) write16(addr uint16, val uint16) {
	b.write8(addr, uint8(val&0xFF))
	b.write8(addr+1, uint8(val&0xFF00)>>8)
}

func (b *bus) init() {
	// CPU, PPU and DMA mappers and APU
	b.maps = make([]BusMapInt, 4)
}

func (b *bus) connect(mapId int, busInt busInt) {
	b.maps[mapId].busInt = busInt
}

func (b *bus) getBusInt(mapId int) *BusMapInt {
	return &b.maps[mapId]
}
