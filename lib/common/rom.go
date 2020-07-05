package common

import (
	"crypto/md5"
	"io"
	"os"
)

//	busInt
type Rom struct {
	rom []byte

	writable bool
}

func (r *Rom) Read8(addr uint16) uint8 {
	return r.rom[addr]
}
func (r *Rom) Read8w(addr uint32) uint8 {
	return r.rom[addr]
}

// little endian
func (r *Rom) Read16(addr uint16) uint16 {
	return uint16(r.Read8(addr)) | uint16(r.Read8(addr+1))<<8
}
func (r *Rom) Write8(addr uint16, val uint8) {
	r.Write8w(uint32(addr), val)
}
func (r *Rom) Write8w(addr uint32, val uint8) {
	if r.writable {
		r.rom[addr] = val
	} else {
		panic("Rom is not writable")
	}
}
func (r *Rom) Write16(addr uint16, val uint16) {
	if r.writable {
		r.Write8(addr, uint8(val&0xFF))
		r.Write8(addr+1, uint8((val&0xFF00)>>8))
	} else {
		panic("Rom is not writable")
	}
}

func (r *Rom) Size() int {
	return len(r.rom)
}

func (r *Rom) Hash() [md5.Size]byte {
	return md5.Sum(r.rom)
}

func (r *Rom) Init(size int, writable bool) {
	r.rom = make([]byte, size, size)
	r.writable = writable
}
func (r *Rom) LoadFromFile(file *os.File) (int, error) {
	return io.ReadFull(file, r.rom)
}

// do we even need to since this is rom...?
func (r *Rom) Serialise(s Serialiser) error {
	return s.Serialise(r.rom)
}
func (r *Rom) DeSerialise(s Serialiser) error {
	return s.DeSerialise(&r.rom)
}
