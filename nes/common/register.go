package common

import (
	"fmt"
)

type Register struct {
	Val uint8

	name string

	onWrite func()
	onRead  func() uint8
}

type Register16 struct {
	Val uint16

	name string
}

func (r Register) String() string {
	return fmt.Sprintf("%s: 0x%02x", r.name, r.Val)
}
func (r *Register) Init(name string, val uint8) {
	r.Val = val
	r.name = name
}
func (r *Register) Initx(name string, val uint8, onWrite func(), onRead func() uint8) {
	r.Init(name, val)
	r.onWrite = onWrite
	r.onRead = onRead
}
func (r *Register) Set(w uint8) {
	r.Val |= w
}
func (r *Register) Clr(w uint8) {
	r.Val &= w ^ 0xFF
}

func (r *Register) Write(w uint8) {
	r.Val = w

	if r.onWrite != nil {
		r.onWrite()
	}
}
func (r *Register) Read() uint8 {
	// add logging so we can debug it, same to Write actually
	// where to control logging level without having to propagate a flag to each component,
	// have a package level?? probably ok
	if r.onRead != nil {
		return r.onRead()
	}
	return r.Val
}

func (r Register16) String() string {
	return fmt.Sprintf("%s: 0x%04x", r.name, r.Val)
}
func (r *Register16) Init(name string, val uint16) {
	r.Val = val
	r.name = name
}
func (r *Register16) Write(w uint16) {
	r.Val = w
}
func (r *Register16) Read() uint16 {
	return r.Val
}
