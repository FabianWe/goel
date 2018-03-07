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

import (
	"fmt"
	"strings"
)

// AbstractLiteral is the base interface for all concrete domain literals.
// Such literals are for example floats or strings.
// Abstract literals should be comparable and values from two different domains
// are never equal. That is checked in go automatically, from the specification:
//
// "Interface values are comparable. Two interface values are equal if they have
// identical dynamic types and equal dynamic values or if both have value nil."
//
// Source: https://golang.org/ref/spec#Comparison_operators
type AbstractLiteral interface {
}

// Predicate is a predicate such as > for a concrete domain: It has an arity
// n > 0 and an extension (a realation with that arity).
// Enumerating all values is not possible (for example the smaller relation on
// ℚ), thus it is defined as a function that takes n arguments and returns true
// or false.
// A predicate relation may assume that it is only called with exactly n
// elements (n is returned by the Arity() function).
//
// It may also assume that all values are indeed part of the concrete domain,
// see ConcreteDomain type for more details.
type Predicate interface {
	Arity() int
	Relation(values ...AbstractLiteral) bool
}

// FeatureID is used to represent different features (functions f1, ... fk)
// as defined in EL++.
type FeatureID int

func NewFeatureID(id int) FeatureID {
	return FeatureID(id)
}

func (id FeatureID) String() string {
	return fmt.Sprintf("f%d", id)
}

// PredicateFormula is a formula of the form p(f1, ..., fk).
type PredicateFormula struct {
	Predicate Predicate
	Features  []FeatureID
}

func NewPredicateFormula(predicate Predicate, features ...FeatureID) *PredicateFormula {
	return &PredicateFormula{Predicate: predicate,
		Features: features}
}

func (f *PredicateFormula) String() string {
	featureStrings := make([]string, len(f.Features))
	for i, feature := range f.Features {
		featureStrings[i] = feature.String()
	}
	joined := strings.Join(featureStrings, ", ")
	return fmt.Sprintf("%v (%s)", f.Predicate, joined)
}

// ConcreteDomain is a concrete domain as defined in EL++. Thus it has a set
// Δ(D) and a set of extensions p(D).
// The set is of course not stored, for example ℚ is an infinite set. We cannot
// even store all predicates (for example predicate >q).
// A concrete domain will have methods to create new relations
// (for example for each q ∈ ℚ); and thus we construct them as needed.
//
// A concrete domain must be able to answer conjunction queries
// (is a set of formulae Γ satisfiable) and answer implication queries (does a
// set of formulae Γ imply another formula).
//
// As mentioned above relations for a concrete domain are created as need be -
// and these relations may here be used in formulae. The concrete domain must
// then understand each relation it has created.
// The details of that process are left to the concrete implementation. The
// functions should panic if they receive a predicate they don't understand.
//
// It also has a function to determine if an abstract literal is part of the
// concrete domain. Concrete domains should document which values are considered
// part of the domain.
//
// Concrete domains must be comparable via ==, that is two instances of a
// concrete domain must be considered equal and == should return false only
// if both sides are not the same concrete domain.
// See remark above about comparing interface values.
type ConcreteDomain interface {
	// Contains checks if an abstract literal is part of the concrete domain.
	// The concrete implementation must document this.
	Contains(l AbstractLiteral) bool
	// ConjSat must check if the conjuntion of all formulae is satisfiable.
	// Each formula consists of an id (the predicate as returned by GetPredicates)
	// and all the features (that become the variables in the first-order
	// formula).
	ConjSat(gamma ...*PredicateFormula) bool
	// Implies must check if the conjunction of formulae Γ implies the formula.
	Implies(formula *PredicateFormula, gamma ...*PredicateFormula) bool

	Name() string
}

//go:generate stringer -type=ConcreteDomainEnumeration

// ConcreteDomainEnumeration is an enumeration of all concrete domains
// directly available in goel.
// Currently this is only the concrete domain of rationals.
type ConcreteDomainEnumeration int

const (
	Rationals ConcreteDomainEnumeration = iota
)

// I've broken stringer, so here is the string method...

func (dom ConcreteDomainEnumeration) String() string {
	switch dom {
	case Rationals:
		return "Rationals"
	default:
		return fmt.Sprintf("ConcreteDomainEnumeration(%d)", dom)
	}
}

// TypedPredicateFormula is a PredicateFormula that also has information about
// the domain a formula belongs to.
// Domains are identfied by an id, see CDManager for more information.
type TypedPredicateFormula struct {
	Formula  *PredicateFormula
	DomainId ConcreteDomainEnumeration
}

func NewTypedPredicateFormula(formula *PredicateFormula, domain ConcreteDomainEnumeration) *TypedPredicateFormula {
	return &TypedPredicateFormula{formula, domain}
}

// CDManager is used to store all concrete domains and the formulae
// (PredicateFormula) that exist within an ontology.
// It also stores a reference to the domain a formula was created for.
// It uses uints to identify the formulae, with indices from 0 to n-1
// where n is the number of formulae.
// Thus it just stores TypedPredicateFormula instances.
// The Domains are normally fixed and contains all domains implemented.
// That is at the moment only the Rationals (ℚ).
// There also exists a mapping to get all formulae for a given domain.
//
// Addming formulae is usually done with the build-in Add method, this will take
// care that the mapping from domain ↦ formulae of domain is kept in tact.
// Note that this method is not safe for concurrent use.
type CDManager struct {
	Domains  []ConcreteDomain
	Formulae []*TypedPredicateFormula

	// domainMap map[ConcreteDomainEnumeration][]*TypedPredicateFormula
	domainMap [][]*TypedPredicateFormula
}

func NewCDManager() *CDManager {
	domains := []ConcreteDomain{NewRationalDomain()}
	domainMap := [][]*TypedPredicateFormula{[]*TypedPredicateFormula{}}
	return &CDManager{domains, nil, domainMap}
}

func (m *CDManager) AddDomain(d ConcreteDomain) {
	m.Domains = append(m.Domains, d)
	m.domainMap = append(m.domainMap, []*TypedPredicateFormula{})
}

func (m *CDManager) GetDomainByID(id ConcreteDomainEnumeration) ConcreteDomain {
	return m.Domains[id]
}

// Add adds a new formula to the manager. It updates the Formulae slice (that
// just contains all formulae) and the mapping domain ↦ formulae that contains
// all formulae for a certain domain.
func (m *CDManager) Add(formula *TypedPredicateFormula) {
	m.Formulae = append(m.Formulae, formula)
	m.domainMap[formula.DomainId] = append(m.domainMap[formula.DomainId], formula)
}

// GetFormulaeFor returns all formulae for the given concrete domain. The
// returned slice should not be modified, so just read from it.
func (m *CDManager) GetFormulaeFor(id ConcreteDomainEnumeration) []*TypedPredicateFormula {
	return m.domainMap[id]
}
