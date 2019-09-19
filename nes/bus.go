package gones

type busInt interface {
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

	maps []BusMapInt
}

func (b *BusMapInt) read8(addr uint16) uint8 {

	panic("read8 to missing address range...")
	return 0x0
}
func (b *BusMapInt) read16(addr uint16) uint16 {

	panic("read16 to missing address range...")
	return 0x0
}

func (b *BusMapInt) write8(addr uint16, val uint8) {

	panic("write8 to missing address range...")
}
func (b *BusMapInt) write16(uint16, uint16) {
	panic("write16 not used at the moment...")
}

func (b *bus) init() {
	b.maps = make([]BusMapInt, 2)
}

func (b *bus) getBusInt(mapId int) *BusMapInt {
	return &b.maps[mapId]
}
