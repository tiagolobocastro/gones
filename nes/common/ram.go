package common

import (
	"io"
	"io/ioutil"
	"os"
)

//	busInt
type Ram struct {
	ram []byte
}

func (r *Ram) size() uint16 {
	return uint16(len(r.ram))
}

func (r *Ram) Init(size int) {
	r.ram = make([]byte, size)
}

func (r *Ram) InitNfill(size int, fill uint8) {
	r.Init(size)
	for i := range r.ram {
		r.ram[i] = fill
	}
}

func (r *Ram) Read8(addr uint16) uint8 {
	return r.ram[addr]
}
func (r *Ram) Write8(addr uint16, val uint8) {
	r.ram[addr] = val
}

func (r *Ram) LoadFromFile(file *os.File) (int, error) {
	return io.ReadFull(file, r.ram)
}
func (r *Ram) SaveToFile(file *os.File) error {
	return ioutil.WriteFile(file.Name(), r.ram, 0700)
}

// little endian
func (r *Ram) Read16(addr uint16) uint16 {
	return uint16(r.Read8(addr)) | uint16(r.Read8(addr+1))<<8
}
func (r *Ram) Write16(addr uint16, val uint16) {
	r.Write8(addr, uint8(val&0xFF))
	r.Write8(addr+1, uint8((val&0xFF00)>>8))
}
