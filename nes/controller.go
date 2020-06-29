package gones

import (
	"github.com/tiagolobocastro/gones/nes/common"
)

const (
	bitA = iota
	bitB
	bitSelect
	bitStart
	bitUp
	bitDown
	bitLeft
	bitRight
)

type nesController struct {
	buttons   [8]uint8
	targetBit uint8
}

func (n *nesController) Serialise(s common.Serialiser) error {
	return s.Serialise(n.buttons, n.targetBit)
}
func (n *nesController) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(&n.buttons, &n.targetBit)
}

type controllers struct {
	controllers [2]nesController
	strobe      uint8
}

func (c *controllers) Serialise(s common.Serialiser) error {
	for _, e := range c.controllers {
		e.Serialise(s)
	}
	s.Serialise(c.strobe)
	return nil
}
func (c *controllers) DeSerialise(s common.Serialiser) error {
	for _, e := range c.controllers {
		e.DeSerialise(s)
	}
	s.DeSerialise(&c.strobe)
	return nil
}

func (c *controllers) readButton(controllerId uint8) uint8 {
	controller := &c.controllers[controllerId]

	if controller.targetBit < 8 {
		active := controller.buttons[controller.targetBit]
		controller.targetBit++
		return active
	} else {
		// returns 0 like a non original nes controller :-)
		return 0
	}
}

func (c *controllers) init() {
	c.controllers = [2]nesController{}
	c.strobe = 0
}

func (c *controllers) Reset() {
	c.init()
}

// use interface
func (c *controllers) poke(controllerId uint8, button uint8, pressed bool) {
	// strobing does not really work because we cannot access the "screen"
	// where the control logic is implemented, so it's the screen that pokes us
	controller := &c.controllers[controllerId]
	if pressed {
		controller.buttons[button] = 1
	} else {
		controller.buttons[button] = 0
	}
}

// BusInt
func (c *controllers) Read8(addr uint16) uint8 {

	val := uint8(0)
	switch addr {
	// controller1
	case 0x4016:
		val = c.readButton(0)
	// controller2
	case 0x4017:
		val = c.readButton(1)
	}

	return val
}

func (c *controllers) Write8(addr uint16, val uint8) {
	switch addr {
	case 0x4016:
		// if strobe set start polling the buttons
		// else stop polling
		c.strobe = val & 0x1

		// always or only when clearing the strobe?
		for i := range c.controllers {
			c.controllers[i].targetBit = 0
		}
	}
}
