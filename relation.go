// The MIT License (MIT)
//
// Copyright (c) 2018 Fabian Wenzelmann
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

import "log"

type Relation struct {
	mapping        map[uint]map[uint]struct{}
	reverseMapping map[uint]map[uint]struct{}
}

func NewRelation(initialCapacity uint) *Relation {
	return &Relation{
		mapping:        make(map[uint]map[uint]struct{}, initialCapacity),
		reverseMapping: make(map[uint]map[uint]struct{}, initialCapacity),
	}
}

func addToRelation(m map[uint]map[uint]struct{}, first, second uint) bool {
	inner, has := m[first]
	if !has {
		inner = make(map[uint]struct{})
		m[first] = inner
	}
	// TODO correct? should be...
	oldLen := len(inner)
	inner[second] = struct{}{}
	return len(inner) != oldLen
}

func (r *Relation) Add(c, d uint) bool {
	// TODO simplify when tested.
	first := addToRelation(r.mapping, c, d)
	second := addToRelation(r.reverseMapping, d, c)
	if first != second {
		log.Printf("Unexpected Relation behaviour while adding %d --> %d: Mappings not consistent",
			c, d)
		return false
	}
	return first
}

func (r *Relation) Contains(c, d uint) bool {
	inner, hasInner := r.mapping[c]
	if !hasInner {
		return false
	}
	_, has := inner[d]
	return has
}
