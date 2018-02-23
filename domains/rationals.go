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

package domains

import "math"

// RationalDomain is the concrete domain for rational numbers (ℚ).
// Only elements of type float64 are considered part of this domain.
//
// It has the following predicates:
//
// (1) A unary predicate ⊤(ℚ) (contains all q ∈ ℚ)
//
// (2) A unary predicate =(q) for all q ∈ ℚ (contains itself)
//
// (3) A unary predicate >(q) for all q ∈ ℚ (contains all elements that are
// greater than q)
//
// (4) A binary predicate +(q) for all q ∈ ℚ with the semantics
// { (q', q'') ∈ ℚ² | q' + q = q'' }
type RationalDomain struct{}

func (d RationalDomain) Contains(l AbstractLiteral) bool {
	_, res := l.(float64)
	return res
}

// TrueRational implements the domain ⊤(ℚ).
type TrueRational struct{}

func NewTrueRational() TrueRational {
	return TrueRational{}
}

func (r TrueRational) Arity() int {
	return 1
}

func (r TrueRational) Relation(values ...AbstractLiteral) bool {
	return true
}

type EqualsRational float64

func NewEqualsRational(q float64) EqualsRational {
	return EqualsRational(q)
}

func (r EqualsRational) Arity() int {
	return 1
}

// TODO hope the compare is correct that way...
func (r EqualsRational) Relation(values ...AbstractLiteral) bool {
	qPrime := values[0].(float64)
	return float64(r) == qPrime
}

type EqualsRationalWithEpsilon struct {
	Epsilon, Q float64
}

func NewEqualsRationalWithEpsilon(epsilon, q float64) EqualsRationalWithEpsilon {
	return EqualsRationalWithEpsilon{
		Epsilon: epsilon,
		Q:       q,
	}
}

func (r EqualsRationalWithEpsilon) Arity() int {
	return 1
}

func (r EqualsRationalWithEpsilon) Relation(values ...AbstractLiteral) bool {
	qPrime := values[0].(float64)
	return math.Abs(qPrime-r.Q) <= r.Epsilon
}

type GreaterRational float64

func NewGreaterRational(q float64) GreaterRational {
	return GreaterRational(q)
}

func (r GreaterRational) Arity() int {
	return 1
}

func (r GreaterRational) Relation(values ...AbstractLiteral) bool {
	qPrime := values[0].(float64)
	return qPrime > float64(r)
}

type BinaryEqualsRational struct{}

func NewBinaryEqualsRational() BinaryEqualsRational {
	return BinaryEqualsRational{}
}

func (r BinaryEqualsRational) Arity() int {
	return 2
}

func (r BinaryEqualsRational) Relation(values ...AbstractLiteral) bool {
	v1 := values[0].(float64)
	v2 := values[1].(float64)
	return v1 == v2
}

type BinaryEqualsRationalWithEpsilon float64

func NewBinaryEqualsRationalWithEpsilon(epsilon float64) BinaryEqualsRationalWithEpsilon {
	return BinaryEqualsRationalWithEpsilon(epsilon)
}

func (r BinaryEqualsRationalWithEpsilon) Airty() int {
	return 2
}

func (r BinaryEqualsRationalWithEpsilon) Relation(values ...AbstractLiteral) bool {
	epsilon := float64(r)
	v1 := values[0].(float64)
	v2 := values[1].(float64)
	return math.Abs(v1-v2) <= epsilon
}

type PlusRational float64

func NewPlusRational(q float64) PlusRational {
	return PlusRational(q)
}

func (r PlusRational) Arity() int {
	return 2
}

func (r PlusRational) Relation(values ...AbstractLiteral) bool {
	q := float64(r)
	q1 := values[0].(float64)
	q2 := values[1].(float64)
	return q1+q == q2
}
