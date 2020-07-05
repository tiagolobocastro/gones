package speakers

import (
	"fmt"
	"sync"
)

type CircularBuffer struct {
	buffer []float64

	// next index to write to
	head int
	// next index to read from
	tail    int
	size    int
	lockSrc sync.Mutex // don't use this one directly
	wait    *sync.Cond

	writeWait chan bool
}

func NewCircularBuffer(size int) *CircularBuffer {
	if size < 2 {
		panic("Invalid size for the CircularBuffer (<2)")
	}
	buffer := CircularBuffer{}
	buffer.reset(size)
	return &buffer
}

func (c *CircularBuffer) Write(value float64, wait bool) error {
	c.wait.L.Lock()
	defer c.wait.L.Unlock()

	if c.isFull() {
		if !wait {
			return fmt.Errorf("buffer is full")
		}
		c.wait.Wait()
	}

	c.buffer[c.head] = value
	c.head = c.getNext(c.head)
	c.wait.Signal()

	return nil
}

func (c *CircularBuffer) ReadInto(slice []float32) int {
	c.wait.L.Lock()
	defer c.wait.L.Unlock()

	if len(slice) > c.available() {
		//go fmt.Print("#")
		return 0
	}

	for i := 0; i < len(slice); i++ {
		slice[i] = float32(c.buffer[c.tail])
		c.tail = c.getNext(c.tail)
	}

	c.wait.Signal()
	return len(slice)
}
func (c *CircularBuffer) ReadInto2(slice [][2]float64) int {
	c.wait.L.Lock()
	defer c.wait.L.Unlock()

	if len(slice) > c.available() {
		//go fmt.Printf("W: %v R: %v F-S: %v N-S: %v N-F: %v\n", writes, reads, fr.Sub(start), out.Sub(start), out.Sub(fr))
		return 0
	}

	for i := 0; i < len(slice); i++ {
		slice[i][0] = c.buffer[c.tail]
		slice[i][1] = c.buffer[c.tail]
		c.tail = c.getNext(c.tail)
	}

	c.wait.Signal()
	return len(slice)
}

func (c *CircularBuffer) Read() (float64, error) {
	c.wait.L.Lock()
	defer c.wait.L.Unlock()

	if c.isEmpty() {
		// we could potentially use an await to wake this up
		// rather than return error?
		return 0, fmt.Errorf("buffer is empty")
	}

	value := c.buffer[c.tail]
	c.tail = c.getNext(c.tail)

	return value, nil
}

// internal helpers
func (c *CircularBuffer) available() int {
	if c.isEmpty() {
		return 0
	}

	if c.head > c.tail {
		return c.head - c.tail - 1
	}

	return c.head + c.size - c.tail
}
func (c *CircularBuffer) Available() int {
	c.wait.L.Lock()
	defer c.wait.L.Unlock()

	return c.available()
}

// Empty because we want to read from tail but
// the head still has not written that index
func (c *CircularBuffer) isEmpty() bool {
	return c.head == c.tail
}

// Full because we want to write to write to head
// but the tail still has not yet read where
// head points to...
// In the our case maybe we can forcefully adjust the tail??
func (c *CircularBuffer) isFull() bool {
	return c.getNext(c.head) == c.tail
}
func (c *CircularBuffer) getNext(index int) int {
	if (index + 1) >= c.size {
		return 0
	}
	return index + 1
}
func (c *CircularBuffer) reset(size int) {
	c.head = 0
	c.tail = 0
	c.size = size
	c.buffer = make([]float64, size)
	c.writeWait = make(chan bool)
	c.wait = sync.NewCond(&c.lockSrc)
}
