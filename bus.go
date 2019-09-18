package gones

import "sort"

type addrRange struct {
	base     uint16
	end      uint16
	nMirrors uint16
}

func (a addrRange) crosses(b addrRange) bool {
	return (a.base >= b.base && a.base <= b.end) || (a.end >= b.base && a.end <= b.end)
}

func (a addrRange) crossover(b addrRange) addrRange {
	addrRange := addrRange{0, 0, 0}
	if a.crosses(b) {
		if a.base > b.base {
			addrRange.base = a.base
		} else {
			addrRange.base = b.base
		}
		if a.end > b.end {
			addrRange.end = b.end
		} else {
			addrRange.end = a.end
		}
	}
	return addrRange
}

type busAddrRange struct {
	addrRange
	size uint16
}
type memMapEntry struct {
	addrRange
	deviceId int
}
type busMemMapEntry struct {
	busAddrRange
	busInt
}

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

	devEntries []busMemMapEntry
}

func (b *BusMapInt) read8(addr uint16) uint8 {
	for _, entry := range b.devEntries {
		last := entry.base + entry.size*(entry.nMirrors+1) - 1
		if addr >= entry.base && addr <= last {
			return entry.busInt.read8(addr - entry.base)
		}
	}
	panic("read8 to missing address range...")
	return 0x0
}
func (b *BusMapInt) read16(addr uint16) uint16 {
	for _, entry := range b.devEntries {
		last := entry.base + entry.size*(entry.nMirrors+1)
		if addr >= entry.base && addr <= last {
			return entry.busInt.read16(addr - entry.base)
		}
	}
	panic("read16 to missing address range...")
	return 0x0
}

func (b *BusMapInt) write8(addr uint16, val uint8) {
	for _, entry := range b.devEntries {
		last := entry.base + entry.size*(entry.nMirrors+1)
		if addr >= entry.base && addr <= last {
			entry.busInt.write8(addr-entry.base, val)
			return
		}
	}
	panic("write8 to missing address range...")
}
func (b *BusMapInt) write16(uint16, uint16) {
	panic("write16 not used at the moment...")
}

func (b *bus) connectMap(mapId int, addrRange addrRange, devEntries []busMemMapEntry) {
	for _, entry := range devEntries {
		// this does not really work with mirroring, just panic for now
		if entry.nMirrors > 0 || addrRange.nMirrors > 0 {
			panic("connectMap does not support ranges with mirrors!")
		}
		if entry.crosses(addrRange) {
			entry.addrRange = entry.crossover(addrRange)
			entry.size = entry.end - entry.base + 1
			b.connect(mapId, &entry)
		}
	}
}

func (b *bus) connect(mapId int, newEntry *busMemMapEntry) {

	if newEntry == nil {
		panic("trying to connect the bus to an invalid device!")
		return
	}

	if newEntry.size == 0 {
		newEntry.size = newEntry.end - newEntry.base + 1
	}

	devMap := b.maps[mapId].devEntries
	devMapLen := len(devMap)

	// make space for the new entry
	devMap = append(devMap, busMemMapEntry{})

	i := sort.Search(devMapLen, func(i int) bool {
		// assumes no addr conflicts as it does not check for the end
		return devMap[i].base > newEntry.base
	})
	copy(devMap[i+1:], devMap[i:])
	devMap[i] = *newEntry
	b.maps[mapId].devEntries = devMap
}

func (b *bus) init() {
	b.maps = make([]BusMapInt, 2)
}

func (b *bus) getBusInt(mapId int) *BusMapInt {
	return &b.maps[mapId]
}
