package gones

import (
	"testing"
)

func Test_newNes(t *testing.T) {
	nes := NewNES(false, "")
	if nes == nil {
		t.Errorf("failed to get nes!")
	}
}

type cpuTest struct {
	prefix  func()
	name    string
	code    string
	result  string
	postfix func()
}

func cmpMem(nes *nes, t *testing.T, checkAddr uint16, expectedVal uint8) {
	checkVal := nes.ram.ram[checkAddr]
	if checkVal != expectedVal {
		t.Errorf("Output of test %s was incorrect!\nGot:\t\t[0x%04x]=%02x\nExpected:\t[0x%04x]=%02x", t.Name(), checkAddr, checkVal, checkAddr, expectedVal)
	}
}

func testCpuTest(nes *nes, t *testing.T, cpuTest cpuTest) {
	nes.reset()
	nes.loadEasyCode(cpuTest.code)
	if cpuTest.prefix != nil {
		cpuTest.prefix()
	}
	nes.cpu.rg.spc.ps.set(bZ|bN, int8(nes.cpu.rg.gp.ac.read()))

	nes.Run()

	if nes.cpu.rg.String() != cpuTest.result+"\n" {
		t.Errorf("[%s][%s] test failed!\nGot:\t\t%s\nExpected:\t%s", t.Name(), cpuTest.name, nes.cpu.rg.String(), cpuTest.result)
	}

	if cpuTest.postfix != nil {
		cpuTest.postfix()
	}
}

// should be able to generate the tests for similar fn's, ld*,st*
func Test_newNes_RunOpTest(t *testing.T) {
	nes := NewNES(false, "")
	if nes == nil {
		t.Fatalf("failed to get nes!")
	}

	var ldaIMM = cpuTest{code: "0600: a9 aa 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0xaa, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, ldaIMM)
	var ldaZPG = cpuTest{code: "0600: a5 bb 00", result: "pc: 0x0603, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x77, x: 0x00, y: 0x00", prefix: func() { nes.ram.ram[0xbb] = 0x77 }}
	testCpuTest(nes, t, ldaZPG)
	var ldaABS = cpuTest{code: "0600: ad 88 18 00", result: "pc: 0x0604, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x88, x: 0x00, y: 0x00", prefix: func() { nes.ram.ram[0x1888] = 0x88 }}
	testCpuTest(nes, t, ldaABS)
	var ldaABX = cpuTest{code: "0600: bd fe ff 00", result: "pc: 0x0604, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x99, x: 0x0d, y: 0x00", prefix: func() {
		nes.ram.ram[0x0B] = 0x99
		nes.cpu.rg.gp.ix.x.write(0xD)
	}}
	testCpuTest(nes, t, ldaABX)
	var ldaABY = cpuTest{code: "0600: b9 fe ff 00", result: "pc: 0x0604, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0xf9, x: 0x00, y: 0x0d", prefix: func() {
		nes.ram.ram[0x0B] = 0xF9
		nes.cpu.rg.gp.ix.y.write(0xD)
	}}
	testCpuTest(nes, t, ldaABY)
	var ldaIIX = cpuTest{code: "0600: a1 00 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0xcc, x: 0x01, y: 0x00", prefix: func() {
		nes.ram.ram[0x2] = 0x1
		nes.ram.ram[0x100] = 0xCC
		nes.cpu.rg.gp.ix.x.write(1)
	}}
	testCpuTest(nes, t, ldaIIX)
	var ldaIIY = cpuTest{code: "0600: b1 01 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0xcc, x: 0x00, y: 0x02", prefix: func() {
		nes.ram.ram[0x2] = 0x1
		nes.ram.ram[0x102] = 0xCC
		nes.cpu.rg.gp.ix.y.write(2)
	}}
	testCpuTest(nes, t, ldaIIY)
	var ldaZPX = cpuTest{code: "0600: b5 ff 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0xfe, x: 0x0b, y: 0x00", prefix: func() {
		nes.ram.ram[0xA] = 0xFE
		nes.cpu.rg.gp.ix.x.write(0xB)
	}}
	testCpuTest(nes, t, ldaZPX)
	var ldxZPY = cpuTest{code: "0600: b6 ff 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0xef, y: 0x0c", prefix: func() {
		nes.ram.ram[0xB] = 0xEF
		nes.cpu.rg.gp.ix.y.write(0xC)
	}}
	testCpuTest(nes, t, ldxZPY)
	var staIIX = cpuTest{code: "0600: 81 21 00", result: "pc: 0x0603, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x0c, x: 0x01, y: 0x00", prefix: func() {
		nes.ram.ram[0x22] = 0x0
		nes.ram.ram[0x23] = 0x1
		nes.cpu.rg.gp.ac.write(0x0C)
		nes.cpu.rg.gp.ix.x.write(1)
	}, postfix: func() {
		cmpMem(nes, t, 0x100, 0x0C)
	}}
	testCpuTest(nes, t, staIIX)
	var staIIY = cpuTest{code: "0600: 91 21 00", result: "pc: 0x0603, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x0c, x: 0x00, y: 0x01", prefix: func() {
		nes.ram.ram[0x21] = 0x10
		nes.ram.ram[0x22] = 0x1
		nes.cpu.rg.gp.ac.write(0x0C)
		nes.cpu.rg.gp.ix.y.write(1)
	}, postfix: func() {
		cmpMem(nes, t, 0x111, 0x0C)
	}}
	testCpuTest(nes, t, staIIY)
	var staZPX = cpuTest{code: "0600: 95 ff 00", result: "pc: 0x0603, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x7e, x: 0x0b, y: 0x00", prefix: func() {
		nes.cpu.rg.gp.ac.write(0x7E)
		nes.cpu.rg.gp.ix.x.write(0xB)
	}, postfix: func() {
		cmpMem(nes, t, 0xA, 0x7E)
	}}
	testCpuTest(nes, t, staZPX)
	var staABY = cpuTest{code: "0600: 99 ff 00 00", result: "pc: 0x0604, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x7e, x: 0x00, y: 0x0b", prefix: func() {
		nes.cpu.rg.gp.ac.write(0x7E)
		nes.cpu.rg.gp.ix.y.write(0xB)
	}, postfix: func() {
		cmpMem(nes, t, 0x010A, 0x7E)
	}}
	testCpuTest(nes, t, staABY)
	var staABX = cpuTest{code: "0600: 9d ff 00 00", result: "pc: 0x0604, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x7f, x: 0x0c, y: 0x00", prefix: func() {
		nes.cpu.rg.gp.ac.write(0x7F)
		nes.cpu.rg.gp.ix.x.write(0xC)
	}, postfix: func() {
		cmpMem(nes, t, 0x010B, 0x7F)
	}}
	testCpuTest(nes, t, staABX)
	var stxZPG = cpuTest{code: "0600: 86 ff 00", result: "pc: 0x0603, sp: 0xff, ps: 0x36 (N:0 V:0 E:1 B:1 D:0 I:1 Z:1 C:0), ac: 0x00, x: 0x0b, y: 0x00", prefix: func() {
		nes.cpu.rg.gp.ix.x.write(0xB)
	}, postfix: func() {
		cmpMem(nes, t, 0xFF, 0x0B)
	}}
	testCpuTest(nes, t, stxZPG)
	var stxABS = cpuTest{code: "0600: 8e 34 02 00", result: "pc: 0x0604, sp: 0xff, ps: 0x36 (N:0 V:0 E:1 B:1 D:0 I:1 Z:1 C:0), ac: 0x00, x: 0x0b, y: 0x00", prefix: func() {
		nes.cpu.rg.gp.ix.x.write(0xB)
	}, postfix: func() {
		cmpMem(nes, t, 0x234, 0x0B)
	}}
	testCpuTest(nes, t, stxABS)
	var stxZPY = cpuTest{code: "0600: 96 34 00", result: "pc: 0x0603, sp: 0xff, ps: 0x36 (N:0 V:0 E:1 B:1 D:0 I:1 Z:1 C:0), ac: 0x00, x: 0x0a, y: 0x08", prefix: func() {
		nes.cpu.rg.gp.ix.x.write(0xa)
		nes.cpu.rg.gp.ix.y.write(0x8)
	}, postfix: func() {
		cmpMem(nes, t, 0x3C, 0xa)
	}}
	testCpuTest(nes, t, stxZPY)
	var ldyIMM = cpuTest{code: "0600: a0 aa 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0x00, y: 0xaa"}
	testCpuTest(nes, t, ldyIMM)
	var ldyZPG = cpuTest{code: "0600: a4 bb 00", result: "pc: 0x0603, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0x00, y: 0x77", prefix: func() { nes.ram.ram[0xbb] = 0x77 }}
	testCpuTest(nes, t, ldyZPG)
	var ldyABS = cpuTest{code: "0600: ac 88 18 00", result: "pc: 0x0604, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0x00, y: 0x88", prefix: func() { nes.ram.ram[0x1888] = 0x88 }}
	testCpuTest(nes, t, ldyABS)
	var ldyABX = cpuTest{code: "0600: bc fe ff 00", result: "pc: 0x0604, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0x0d, y: 0x99", prefix: func() {
		nes.ram.ram[0x0B] = 0x99
		nes.cpu.rg.gp.ix.x.write(0xD)
	}}
	testCpuTest(nes, t, ldyABX)
	var ldyZPX = cpuTest{code: "0600: b4 ff 00", result: "pc: 0x0603, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x00, x: 0x0b, y: 0xfe", prefix: func() {
		nes.ram.ram[0xA] = 0xFE
		nes.cpu.rg.gp.ix.x.write(0xB)
	}}
	testCpuTest(nes, t, ldyZPX)
}

func Test_JMP(t *testing.T) {
	nes := NewNES(false, "")
	if nes == nil {
		t.Fatalf("failed to get nes!")
	}

	var jmpABS = cpuTest{code: "0600: a9 01 4c 07 06 a9 22 00", result: "pc: 0x0608, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x01, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, jmpABS)
	var jmpIND = cpuTest{code: "0600: a9 0e 8d f0 00 a9 06 8d f1 00 6c f0 00 00 a9 22", result: "pc: 0x0611, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x22, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, jmpIND)
	var jmpINDBug = cpuTest{code: "0600: a9 0e 8d ff 01 a9 06 8d 00 01 6c ff 01 00 a9 22", result: "pc: 0x0611, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x22, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, jmpINDBug)
	var bpl = cpuTest{code: "0600: a9 81 10 03 a9 22 00 a9 33", result: "pc: 0x0607, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x22, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, bpl)
	var bplFw = cpuTest{code: "0600: a9 51 10 03 a9 22 00 a9 33", result: "pc: 0x060a, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x33, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, bplFw)
	var bplBw = cpuTest{code: "0600: 4c 06 06 a9 33 00 a9 51 10 f9 a9 44 00", result: "pc: 0x0606, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x33, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, bplBw)
	var bmi = cpuTest{code: "0600: a9 51 30 03 a9 22 00 a9 33", result: "pc: 0x0607, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x22, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, bmi)
	var jsrRts = cpuTest{code: "0600: 20 04 06 00 a9 11 60", result: "pc: 0x0604, sp: 0xff, ps: 0x34 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x11, x: 0x00, y: 0x00"}
	testCpuTest(nes, t, jsrRts)
}

func Test_LA(t *testing.T) {
	nes := NewNES(false, "")
	if nes == nil {
		t.Fatalf("failed to get nes!")
	}

	tests := []cpuTest{
		{name: "sbcIMM", code: "0600: 18 a9 fe e9 7e 00", result: "pc: 0x0606, sp: 0xff, ps: 0x75 (N:0 V:1 E:1 B:1 D:0 I:1 Z:0 C:1), ac: 0x7f, x: 0x00, y: 0x00"},
		{name: "sbcIMM2", code: "0600: 18 a9 fe e9 7d 00", result: "pc: 0x0606, sp: 0xff, ps: 0xb5 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:1), ac: 0x80, x: 0x00, y: 0x00"},
		{name: "sbcIMM3", code: "0600: a9 fe e9 7e 00", result: "pc: 0x0605, sp: 0xff, ps: 0x75 (N:0 V:1 E:1 B:1 D:0 I:1 Z:0 C:1), ac: 0x7f, x: 0x00, y: 0x00"},

		{name: "cmpIMM", code: "0600: a9 03 c9 05 00", result: "pc: 0x0605, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x03, x: 0x00, y: 0x00"},
		{name: "cmpIMM2", code: "0600: a9 03 c9 03 00", result: "pc: 0x0605, sp: 0xff, ps: 0x37 (N:0 V:0 E:1 B:1 D:0 I:1 Z:1 C:1), ac: 0x03, x: 0x00, y: 0x00"},
		{name: "cmpIMM3", code: "0600: a9 03 c9 01 00", result: "pc: 0x0605, sp: 0xff, ps: 0x35 (N:0 V:0 E:1 B:1 D:0 I:1 Z:0 C:1), ac: 0x03, x: 0x00, y: 0x00"},
		// below actually revealed a bug in easy6502 online as it sets C!
		{name: "cmpIMM4", code: "0600: a9 85 c9 01 00", result: "pc: 0x0605, sp: 0xff, ps: 0xb4 (N:1 V:0 E:1 B:1 D:0 I:1 Z:0 C:0), ac: 0x85, x: 0x00, y: 0x00"},
	}

	for _, test := range tests {
		testCpuTest(nes, t, test)
	}
}
