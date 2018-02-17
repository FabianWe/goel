// The MIT License (MIT)
//
// Copyright (c) 2016, 2017, 2018 Fabian Wenzelmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package goel

type BCSet struct {
	m map[uint]struct{}
	c *ELBaseComponents
}

func NewBCSet(c *ELBaseComponents, initialCapacity uint) *BCSet {
	return &BCSet{
		m: make(map[uint]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCSet) Contains(c Concept) bool {
	_, has := s.m[c.NormalizedID(s.c)]
	return has
}

func (s *BCSet) ContainsID(c uint) bool {
	_, has := s.m[c]
	return has
}

// TODO sometimes we get null here... check again where it is used.
func (s *BCSet) Add(c Concept) bool {
	oldLen := len(s.m)
	s.m[c.NormalizedID(s.c)] = struct{}{}
	return oldLen != len(s.m)
}

func (s *BCSet) AddID(c uint) bool {
	oldLen := len(s.m)
	s.m[c] = struct{}{}
	return oldLen != len(s.m)
}

func (s *BCSet) Union(other *BCSet) bool {
	oldLen := len(s.m)
	for v, _ := range other.m {
		s.m[v] = struct{}{}
	}
	return oldLen != len(s.m)
}

// IsSubset tests if s ⊆ other.
func (s *BCSet) IsSubset(other *BCSet) bool {
	if len(s.m) > len(other.m) {
		return false
	}
	for v, _ := range s.m {
		if _, hasD := other.m[v]; !hasD {
			return false
		}
	}
	return true
}

// Equals checks if s = other.
func (s *BCSet) Equals(other *BCSet) bool {
	return len(s.m) == len(other.m) && s.IsSubset(other)
}

func (s *BCSet) Copy() *BCSet {
	res := NewBCSet(s.c, uint(len(s.m)))
	for v, _ := range s.m {
		res.m[v] = struct{}{}
	}
	return res
}

type bcPair struct {
	First, Second uint
}

type BCPairSet struct {
	M map[bcPair]struct{}
	c *ELBaseComponents
}

func NewBCPairSet(c *ELBaseComponents, initialCapacity uint) *BCPairSet {
	return &BCPairSet{
		M: make(map[bcPair]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCPairSet) Contains(c, d Concept) bool {
	_, has := s.M[bcPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}]
	return has
}

func (s *BCPairSet) ContainsID(c, d uint) bool {
	_, has := s.M[bcPair{c, d}]
	return has
}

func (s *BCPairSet) Add(c, d Concept) bool {
	oldLen := len(s.M)
	p := bcPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}
	s.M[p] = struct{}{}
	return oldLen != len(s.M)
}

func (s *BCPairSet) AddID(c, d uint) bool {
	oldLen := len(s.M)
	p := bcPair{c, d}
	s.M[p] = struct{}{}
	return oldLen != len(s.M)
}
