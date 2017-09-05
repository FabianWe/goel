// The MIT License (MIT)

// Copyright (c) 2016, 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package goel

import "sync"

// IntMax returns the maximum of a and b.
func IntMax(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

// IntMin returns the minimum of a and b.
func IntMin(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

// UIntMax returns the maximum of a and b.
func UIntMax(a, b uint) uint {
	if a > b {
		return a
	} else {
		return b
	}
}

// UIntMin returns the minimum of a and b.
func UIntMin(a, b uint) uint {
	if a < b {
		return a
	} else {
		return b
	}
}

// IntDistributor is a type used to generate new uint values.
// The solving algorithm requires that new names are introduced, we use this
// distributor to generate new integer values concurrently.
// This means that the Next method can be used concurrently from different
// goroutines.
type IntDistributor struct {
	next  uint
	mutex *sync.Mutex
}

// NewIntDistributor returns a new distributor s.t. the Next method first
// produces the value next.
func NewIntDistributor(next uint) *IntDistributor {
	return &IntDistributor{next: next, mutex: new(sync.Mutex)}
}

// Next returns the next integer value. That is the first element produced
// is the provided next value, then next + 1 etc.
//
// It is safe to use concurrently from different goroutines.
func (dist *IntDistributor) Next() uint {
	dist.mutex.Lock()
	defer dist.mutex.Unlock()
	next := dist.next
	dist.next++
	return next
}
