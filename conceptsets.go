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

import (
	"github.com/FabianWe/goel/domains"
)

type BCSet struct {
	M map[uint]struct{}
	c *ELBaseComponents
}

func NewBCSet(c *ELBaseComponents, initialCapacity uint) *BCSet {
	return &BCSet{
		M: make(map[uint]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCSet) String() string {
	return StringUintSet(s.M)
}

func (s *BCSet) Contains(c Concept) bool {
	_, has := s.M[c.NormalizedID(s.c)]
	return has
}

func (s *BCSet) ContainsID(c uint) bool {
	_, has := s.M[c]
	return has
}

// TODO sometimes we get null here... check again where it is used.
func (s *BCSet) Add(c Concept) bool {
	// TODO remove debug
	oldLen := len(s.M)
	s.M[c.NormalizedID(s.c)] = struct{}{}
	return oldLen != len(s.M)
}

func (s *BCSet) AddID(c uint) bool {
	oldLen := len(s.M)
	s.M[c] = struct{}{}
	return oldLen != len(s.M)
}

func (s *BCSet) Union(other *BCSet) bool {
	oldLen := len(s.M)
	for v, _ := range other.M {
		s.M[v] = struct{}{}
	}
	return oldLen != len(s.M)
}

// IsSubset tests if s ⊆ other.
func (s *BCSet) IsSubset(other *BCSet) bool {
	if len(s.M) > len(other.M) {
		return false
	}
	for v, _ := range s.M {
		if _, hasD := other.M[v]; !hasD {
			return false
		}
	}
	return true
}

// Equals checks if s = other.
func (s *BCSet) Equals(other *BCSet) bool {
	return len(s.M) == len(other.M) && s.IsSubset(other)
}

func (s *BCSet) Copy() *BCSet {
	res := NewBCSet(s.c, uint(len(s.M)))
	for v, _ := range s.M {
		res.M[v] = struct{}{}
	}
	return res
}

func (s *BCSet) GetCDConjunction(manager *domains.CDManager) [][]*domains.PredicateFormula {
	res := make([][]*domains.PredicateFormula, len(manager.Domains))
	for candidate, _ := range s.M {
		concept := s.c.GetConcept(candidate)
		cdExt, ok := concept.(ConcreteDomainExtension)
		if !ok {
			continue
		}
		// now look up the formula via the manager
		formula := manager.GetFormulaByID(uint(cdExt))
		// lookup the domain and add result
		res[formula.DomainId] = append(res[formula.DomainId], formula.Formula)
	}
	return res
}

func CompareSMapping(first, second []*BCSet) bool {
	if len(first) != len(second) {
		return false
	}
	n := len(first)
	for i := 1; i < n; i++ {
		if !first[i].Equals(second[i]) {
			return false
		}
	}
	return true
}

type BCPair struct {
	First, Second uint
}

type BCPairSet struct {
	M map[BCPair]struct{}
	c *ELBaseComponents
}

func NewBCPairSet(c *ELBaseComponents, initialCapacity uint) *BCPairSet {
	return &BCPairSet{
		M: make(map[BCPair]struct{}, initialCapacity),
		c: c,
	}
}

func (s *BCPairSet) Contains(c, d Concept) bool {
	_, has := s.M[BCPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}]
	return has
}

func (s *BCPairSet) ContainsID(c, d uint) bool {
	_, has := s.M[BCPair{c, d}]
	return has
}

func (s *BCPairSet) Add(c, d Concept) bool {
	oldLen := len(s.M)
	p := BCPair{c.NormalizedID(s.c), d.NormalizedID(s.c)}
	s.M[p] = struct{}{}
	return oldLen != len(s.M)
}

func (s *BCPairSet) AddID(c, d uint) bool {
	oldLen := len(s.M)
	p := BCPair{c, d}
	s.M[p] = struct{}{}
	return oldLen != len(s.M)
}
