package common

type BusInt interface {
	// Data Operations
	Read8(uint16) uint8
	Write8(uint16, uint8)
}

type BusExtInt interface {
	// Data Operations
	Read8(uint16) uint8
	Write8(uint16, uint8)
	Read16(uint16) uint16
	Write16(uint16, uint16)
}

type Bus struct {
	//	BusInt
	maps []BusMapInt
}

type BusMapInt struct {
	//	BusInt
	mapId uint

	BusInt
}

func (b *BusMapInt) Read8(addr uint16) uint8 {
	return b.BusInt.Read8(addr)
}
func (b *BusMapInt) Read16(addr uint16) uint16 {
	return uint16(b.Read8(addr)) | uint16(b.Read8(addr+1))<<8
}

func (b *BusMapInt) Write8(addr uint16, val uint8) {
	b.BusInt.Write8(addr, val)
}
func (b *BusMapInt) Write16(addr uint16, val uint16) {
	b.Write8(addr, uint8(val&0xFF))
	b.Write8(addr+1, uint8((val&0xFF00)>>8))
}

func (b *Bus) Init() {
	// CPU, PPU and DMA mappers and APU
	b.maps = make([]BusMapInt, 4)
}

func (b *Bus) Connect(mapId int, busInt BusInt) {
	b.maps[mapId].BusInt = busInt
}

func (b *Bus) GetBusInt(mapId int) *BusMapInt {
	return &b.maps[mapId]
}
