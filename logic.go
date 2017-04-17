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

import (
	"fmt"
	"strings"
)

//// Concepts ////

// ELBaseComponents used to save basic information about an EL++ instance.
// It stores only the number of each base concept, nothing else.
type ELBaseComponents struct {
	Nominals     uint
	CDExtensions uint
	Names        uint
	Roles        uint
}

// NewELBaseComponents returns a new ELBaseComponents instance.
func NewELBaseComponents(nominals, cdExtensions, names, roles uint) *ELBaseComponents {
	return &ELBaseComponents{Nominals: nominals, CDExtensions: cdExtensions,
		Names: names, Roles: roles}
}

// Concept is the interface for all Concept defintions.
// Concepts in EL++ are defined recursively, this is the general interface.
type Concept interface {
}

// TopConcept is the Top concept ⊤.
type TopConcept struct{}

// NewTopConcept returns a new TopConcept.
// Instead of creating it again and again all the time you should
// use the const value Top.
func NewTopConcept() TopConcept {
	return TopConcept{}
}

func (top TopConcept) String() string {
	return "⊤"
}

// BottomConcept is the Bottom Concept ⊥.
type BottomConcept struct{}

// NewBottomConcept returns a new BottomConcept.
// Instead of creating it again and again all the time you should
// use the const value Bottom.
func NewBottomConcept() BottomConcept {
	return BottomConcept{}
}

func (bot BottomConcept) String() string {
	return "⊥"
}

// Top is a constant concept that represents to top concept ⊤.
var Top TopConcept = NewTopConcept()

// Bottom is a constant concept that represents the bottom concept ⊥.
var Bottom BottomConcept = NewBottomConcept()

// NamedConcept is a concept from the set of concept names, identified by an
// id.
// Each concept name A ∈ N_C is encoded as a unique integer with this type.
type NamedConcept uint

func (name NamedConcept) String() string {
	return fmt.Sprintf("A(%d)", name)
}

// Nominal is a nominal a ∈ N_I, identified by id.
// Each nominal a ∈ N_I is encoded as a unique integer with this type.
type Nominal uint

func (nominal Nominal) String() string {
	return fmt.Sprintf("a(%d)", nominal)
}

// NominalConcept is a nominal concept of the form {a}, identified by id.
// A Nominal is encoded by the type Nominal, a NominalConcept is just the usage
// of a Nominal as a Concept.
type NominalConcept Nominal

func (nominal NominalConcept) String() string {
	return fmt.Sprintf("{a(%d)}", nominal)
}

// Role is an EL++ role r ∈ N_R, identifiey by id.
// Each r ∈ N_R is encoded as a unique integer with this type.
type Role uint

func (role Role) String() string {
	return fmt.Sprintf("r(%d)", role)
}

// ConcreteDomainExtension is a concrete domain extension of the form
// p(f1, ..., fk). All this information (predicate and function) has to be
// stored somewhere else, we only store the an id that identifies the concrete
// domain.
type ConcreteDomainExtension uint

func (cd ConcreteDomainExtension) String() string {
	return fmt.Sprintf("CD(%d)", cd)
}

// Conjunction is a concept of the form C ⊓ D.
type Conjunction struct {
	// C, D are the parts of the conjuntion.
	C, D Concept
}

// NewConjunction returns a new conjunction given C and D.
func NewConjunction(c, d Concept) *Conjunction {
	return &Conjunction{C: c, D: d}
}

func (conjunction *Conjunction) String() string {
	return fmt.Sprintf("(%v ⊓ %v)", conjunction.C, conjunction.D)
}

// ExistentialConcept is a concept of the form ∃r.C.
type ExistentialConcept struct {
	R Role
	C Concept
}

// NewExistentialConcept returns a new existential concept of the form
// ∃r.C.
func NewExistentialConcept(r Role, c Concept) *ExistentialConcept {
	return &ExistentialConcept{R: r, C: c}
}

func (existential *ExistentialConcept) String() string {
	return fmt.Sprintf("∃ %v.%v", existential.R, existential.C)
}

//// TBox ////

// GCIConstraint is a general concept inclusion of the form C ⊑ D.
type GCIConstraint struct {
	C, D Concept
}

// NewGCIConstraint returns a new general concept inclusion C ⊑ D.
func NewGCIConstraint(c, d Concept) *GCIConstraint {
	return &GCIConstraint{C: c, D: d}
}

func (gci *GCIConstraint) String() string {
	return fmt.Sprintf("%v ⊑ %v", gci.C, gci.D)
}

// RoleInclusion is a role inclusion of the form r1 o ... o rk ⊑ r.
type RoleInclusion struct {
	// LHS contains the left-hand side r1 o ... o rk.
	LHS []Role
	// RHS is the right-hand side r.
	RHS Role
}

// NewRoleInclusion returns a new role inclusion r1 o ... o rk ⊑ r.
func NewRoleInclusion(lhs []Role, rhs Role) *RoleInclusion {
	return &RoleInclusion{LHS: lhs, RHS: rhs}
}

func (ri *RoleInclusion) String() string {
	// To use the strings.Join function we first generate a slice of the roles
	// as string
	strs := make([]string, len(ri.LHS))
	for i, r := range ri.LHS {
		strs[i] = r.String()
	}
	return fmt.Sprintf("%s ⊑ %s", strings.Join(strs, "o"), ri.RHS.String())
}

// TBox describes a TBox as a set of GCIs and RIs.
type TBox struct {
	GCIs []*GCIConstraint
	RIs  []*RoleInclusion
}

// NewTBox returns a new TBox.
func NewTBox(gcis []*GCIConstraint, ris []*RoleInclusion) *TBox {
	return &TBox{GCIs: gcis, RIs: ris}
}

//// ABox ////

// ConceptAssertion is an concept assertion of the form C(a).
type ConceptAssertion struct {
	C Concept
	A Nominal
}

// NewConceptAssertion returns a new concept assertion C(a).
func NewConceptAssertion(c Concept, a Nominal) *ConceptAssertion {
	return &ConceptAssertion{C: c, A: a}
}

func (ca *ConceptAssertion) String() string {
	return fmt.Sprintf("%v(%v)", ca.C, ca.A)
}

// RoleAssertion is a role assertion of the form r(a, b).
type RoleAssertion struct {
	R    Role
	A, B Nominal
}

// NewRoleAssertion returns a new role assertion r(a, b).
func NewRoleAssertion(r Role, a, b Nominal) *RoleAssertion {
	return &RoleAssertion{R: r, A: a, B: b}
}

func (ra *RoleAssertion) String() string {
	return fmt.Sprintf("%v(%v, %v)", ra.R, ra.A, ra.B)
}

// ABox describes an ABox as a set of concept assertions and role assertions.
type ABox struct {
	ConceptAssertions []*ConceptAssertion
	RoleAssertions    []*RoleAssertion
}

// NewABox returns a new ABox.
func NewABox(conceptAssertions []*ConceptAssertion, roleAssertions []*RoleAssertion) *ABox {
	return &ABox{ConceptAssertions: conceptAssertions, RoleAssertions: roleAssertions}
}
