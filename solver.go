// The MIT License (MIT)
//
// Copyright (c) 2016, 2017 Fabian Wenzelmann
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

type BCSet interface {
	Contains(c Concept) bool
	Add(c Concept) bool
}

type BCSetFactory func() BCSet

type MapBCSet struct {
	m map[uint]struct{}
	c *ELBaseComponents
}

func NewMapBCSet(c *ELBaseComponents, initialCapacity uint) *MapBCSet {
	return &MapBCSet{
		m: make(map[uint]struct{}, initialCapacity),
		c: c,
	}
}

func MapBCSetFactory(c *ELBaseComponents, initialCapacity uint) BCSetFactory {
	return func() BCSet {
		return NewMapBCSet(c, initialCapacity)
	}
}

func (s *MapBCSet) Contains(c Concept) bool {
	_, has := s.m[c.NormalizedID(s.c)]
	return has
}

func (s *MapBCSet) Add(c Concept) bool {
	normalizedID := c.NormalizedID(s.c)
	if _, has := s.m[normalizedID]; has {
		return false
	} else {
		s.m[normalizedID] = struct{}{}
		return true
	}
}

type BCPairSet interface {
	Contains(c, d Concept) bool
	Add(c, d Concept) bool
}

type BCPairSetFactory func() BCPairSet

type bcPair struct {
	First, Second uint
}

func newBCPair(c, d Concept, comp *ELBaseComponents) bcPair {
	return bcPair{
		First:  c.NormalizedID(comp),
		Second: d.NormalizedID(comp),
	}
}

type MapBCPairSet struct {
	m map[bcPair]struct{}
	c *ELBaseComponents
}

func NewMapBCPairSet(c *ELBaseComponents, initialCapacity uint) *MapBCPairSet {
	return &MapBCPairSet{
		m: make(map[bcPair]struct{}, initialCapacity),
		c: c,
	}
}

func MapBCPairSetFactory(c *ELBaseComponents, initialCapacity uint) BCPairSetFactory {
	return func() BCPairSet {
		return NewMapBCPairSet(c, initialCapacity)
	}
}

func (s *MapBCPairSet) Contains(c, d Concept) bool {
	_, has := s.m[newBCPair(c, d, s.c)]
	return has
}

func (s *MapBCPairSet) Add(c, d Concept) bool {
	p := newBCPair(c, d, s.c)
	if _, has := s.m[p]; has {
		return false
	}
	s.m[p] = struct{}{}
	return true
}

type BCMap interface {
	GetVAlue(c Concept) BCSet
}

type DefaultBCMap struct {
	m       map[uint]BCSet
	factory BCSetFactory
	c       *ELBaseComponents
}

func NewDefaultBCMap(factory BCSetFactory, c *ELBaseComponents, initialCapacity uint) *DefaultBCMap {
	return &DefaultBCMap{
		m:       make(map[uint]BCSet, initialCapacity),
		factory: factory,
		c:       c,
	}
}

func (m *DefaultBCMap) GetValue(c Concept) BCSet {
	id := c.NormalizedID(m.c)
	if res, has := m.m[id]; has {
		return res
	} else {
		newSet := m.factory()
		m.m[id] = newSet
		return newSet
	}
}

type RoleMap interface {
	GetValue(r Role) BCPairSet
}

type DefaultRoleMap struct {
	m       map[Role]BCPairSet
	factory BCPairSetFactory
}

func NewDefaultRoleMap(factory BCPairSetFactory, initialCapacity uint) *DefaultRoleMap {
	return &DefaultRoleMap{
		m:       make(map[Role]BCPairSet, initialCapacity),
		factory: factory,
	}
}

func (m *DefaultRoleMap) GetValue(r Role) BCPairSet {
	if res, has := m.m[r]; has {
		return res
	} else {
		newSet := m.factory()
		m.m[r] = newSet
		return newSet
	}
}
