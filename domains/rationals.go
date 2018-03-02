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
	"log"
	"reflect"

	"github.com/draffensperger/golp"
)

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
//
// Answering the logical queries is done by solving a linear program.
type RationalDomain struct{}

func NewRationalDomain() RationalDomain {
	return RationalDomain{}
}

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

func (r TrueRational) String() string {
	return "⊤(ℚ)"
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

func (r EqualsRational) String() string {
	return fmt.Sprintf("=q(%f)", r)
}

// type EqualsRationalWithEpsilon struct {
// 	Epsilon, Q float64
// }
//
// func NewEqualsRationalWithEpsilon(epsilon, q float64) EqualsRationalWithEpsilon {
// 	return EqualsRationalWithEpsilon{
// 		Epsilon: epsilon,
// 		Q:       q,
// 	}
// }
//
// func (r EqualsRationalWithEpsilon) Arity() int {
// 	return 1
// }
//
// func (r EqualsRationalWithEpsilon) Relation(values ...AbstractLiteral) bool {
// 	qPrime := values[0].(float64)
// 	return math.Abs(qPrime-r.Q) <= r.Epsilon
// }

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

func (r GreaterRational) String() string {
	return fmt.Sprintf(">q(%f)", r)
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

func (r BinaryEqualsRational) String() string {
	return fmt.Sprintf("fi =ℚ fj")
}

type lessRational float64

func newLessRational(q float64) lessRational {
	return lessRational(q)
}

func (r lessRational) Arity() int {
	return 1
}

func (r lessRational) Relation(values ...AbstractLiteral) bool {
	qPrime := values[0].(float64)
	return qPrime < float64(r)
}

type binaryLessRational struct{}

func newBinaryLessRational() binaryLessRational {
	return binaryLessRational{}
}

func (r binaryLessRational) Arity() int {
	return 2
}

func (r binaryLessRational) Relation(values ...AbstractLiteral) bool {
	v1 := values[0].(float64)
	v2 := values[1].(float64)
	return v1 < v2
}

// type BinaryEqualsRationalWithEpsilon float64
//
// func NewBinaryEqualsRationalWithEpsilon(epsilon float64) BinaryEqualsRationalWithEpsilon {
// 	return BinaryEqualsRationalWithEpsilon(epsilon)
// }
//
// func (r BinaryEqualsRationalWithEpsilon) Airty() int {
// 	return 2
// }
//
// func (r BinaryEqualsRationalWithEpsilon) Relation(values ...AbstractLiteral) bool {
// 	epsilon := float64(r)
// 	v1 := values[0].(float64)
// 	v2 := values[1].(float64)
// 	return math.Abs(v1-v2) <= epsilon
// }

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

// Reasoning in ℚ

// LPEpsilon is a value used for solving linear programs with lp_solve.
// lp_solve makes no distinction between > and >=, it's treated as the same
// relation. Thus if we have f1 > 5.0 and f1 = 5.0 lp_solve will say that this
// is okay.
// Reasoning with linear programs might not be the best approach here, see
// my publication for this problem.
// Here's just an example of how we use this variable:
// Transform a constraint of the form f1 > 5 to f1 > 5 + ε.
// This value should be small but not smaller then the ε uses to check if
// two values are considered equal.
// I'm not very happy with this. I think there are other approaches for simple
// constraint problems of that kind, but implementing them is not really a
// possibility now.
var LPEpsilon float64 = 0.00001

// FormulateLP will formulate the linear program in the rational domain.
// Legal input are all predicates defined by the rational domain, the
// transformation uses LPEpsilon for certain constraints, see publications.
//
// LPSolve must have a objective function set, otherwise it will not consider
// the constraints at all. To do that we simply add a new variable (without
// any constraints) that is only used in the objective function. This way
// lp_solve will consider all the constraints (even if the objective function
// does not depend on it).
// We can't just use an objective function that tries to minimize lets say
// all / a single variable, this may lead to unbound exceptions.
func (d RationalDomain) FormulateLP(gamma ...*PredicateFormula) *golp.LP {
	idMap, numVars := d.getIDMap(gamma...)
	lp := golp.NewLP(0, numVars+1)
	// iterate over each formula and add a constraint
	for _, formula := range gamma {
		switch f := formula.Predicate.(type) {
		case TrueRational:
			// ignore this one, it has no effect on satisfiability
			continue
		case EqualsRational:
			q := float64(f)
			xi := idMap[formula.Features[0]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}}, golp.EQ, q)
		case GreaterRational:
			q := float64(f)
			xi := idMap[formula.Features[0]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}}, golp.GE, q+LPEpsilon)
		case BinaryEqualsRational:
			xi, xj := idMap[formula.Features[0]], idMap[formula.Features[1]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}, {xj, -1}}, golp.EQ, 0)

		case PlusRational:
			q := float64(f)
			xi, xj := idMap[formula.Features[0]], idMap[formula.Features[1]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}, {xj, -1}}, golp.EQ, -q)

		case lessRational:
			q := float64(f)
			xi := idMap[formula.Features[0]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}}, golp.LE, q-LPEpsilon)
		case binaryLessRational:
			xi, xj := idMap[formula.Features[0]], idMap[formula.Features[1]]

			lp.AddConstraintSparse([]golp.Entry{{xi, 1}, {xj, -1}}, golp.LE, -LPEpsilon)
		default:
			panic(fmt.Sprintf("Unknown predicate type for rational domain: %v", reflect.TypeOf(formula.Predicate)))
		}
	}
	// set an objective function
	// our objective function is simple: every value set to 0 except the last
	// (our "cheat variable") set to 1
	// because we also iterate over all variables: remove the lower bound of 0
	// from the lp
	objective := make([]float64, numVars+1)
	for i := 0; i < numVars; i++ {
		objective[i] = 0.0
		lp.SetUnbounded(i)
	}
	objective[numVars] = 1.0
	lp.SetObjFn(objective)
	return lp
}

func (d RationalDomain) getIDMap(gamma ...*PredicateFormula) (map[FeatureID]int, int) {
	res := make(map[FeatureID]int)
	nextID := 0
	for _, formula := range gamma {
		for _, feature := range formula.Features {
			if _, has := res[feature]; !has {
				res[feature] = nextID
				nextID++
			}
		}
	}
	return res, nextID
}

func (d RationalDomain) ConjSat(gamma ...*PredicateFormula) bool {
	// first formulate the lp
	lp := d.FormulateLP(gamma...)
	// try to solve the LP
	sol := lp.Solve()
	switch sol {
	case golp.OPTIMAL, golp.SUBOPTIMAL, golp.FEASFOUND:
		// TODO FEASFOUND correct? documentation is offline right now :(
		// also check NOFEASFOUND
		return true
	case golp.INFEASIBLE:
		return false
	case golp.UNBOUNDED:
		log.Println("Unbounded solution in ConjSat for rational domain, should not happen! Returning false")
		return false
	default:
		log.Println("Unkown solution in ConjSat for rational domain, should not happen! Returning false, result type is", sol)
		return false
	}
}

func (d RationalDomain) Implies(formula PredicateFormula, gamma ...*PredicateFormula) bool {
	// not optimal, we actually build the program twice...
	// but well, at least we do it concurrently
	var additionalOne, additionalTwo *PredicateFormula
	res := make(chan bool, 2)
	switch f := formula.Predicate.(type) {
	case EqualsRational:
		q := float64(f)
		f1 := formula.Features[0]
		r1 := newLessRational(q)
		r2 := NewGreaterRational(q)

		additionalOne = NewPredicateFormula(r1, f1)
		additionalTwo = NewPredicateFormula(r2, f1)
	case GreaterRational:
		q := float64(f)
		f1 := formula.Features[0]
		r1 := newLessRational(q)
		r2 := NewEqualsRational(q)

		additionalOne = NewPredicateFormula(r1, f1)
		additionalTwo = NewPredicateFormula(r2, f1)
	case BinaryEqualsRational:
		f1, f2 := formula.Features[0], formula.Features[1]
		r1 := newBinaryLessRational()
		r2 := newBinaryLessRational()

		additionalOne = NewPredicateFormula(r1, f1, f2)
		additionalTwo = NewPredicateFormula(r2, f2, f1)
	}
	// now we're nearly done... just generate both LPs and solve them
	go func() {
		// no nicer way?
		newGamma := make([]*PredicateFormula, len(gamma), len(gamma)+1)
		copy(newGamma, gamma)
		newGamma = append(newGamma, additionalOne)
		res <- d.ConjSat(newGamma...)
	}()
	go func() {
		newGamma := make([]*PredicateFormula, len(gamma), len(gamma)+1)
		copy(newGamma, gamma)
		newGamma = append(newGamma, additionalTwo)
		res <- d.ConjSat(newGamma...)
	}()
	first := <-res
	second := <-res
	return !first && !second
}

// The following approach is commented out. It was my second draft to solve the
// problems with lp_solve (the first one is not implemented). Because it was so
// much work I feel reluctant to lose everything ;).

// func (d RationalDomain) formulateLP(gamma ...*PredicateFormula) *golp.LP {
// 	// first get the number of variables and the normalized ids of the features
// 	idMap, numVars := d.getIDMap(gamma...)
// 	// we need a preprocessing step as explained above
// 	eq := make(map[FeatureID]float64, numVars)
// 	varEq := make(map[FeatureID]map[FeatureID]struct{}, numVars)
// 	greater := make(map[FeatureID]float64, numVars)
// 	for _, formula := range gamma {
// 		switch f := formula.Predicate.(type) {
// 		case EqualsRational:
// 			q := float64(f)
// 			// check if there is already an entry, if this is the case we're done
// 			// and we can return false if they're not equal
// 			if entry, has := eq[formula.Features[0]]; has {
// 				if entry != q {
// 					return nil
// 				}
// 				// else everything is ok, was there twice
// 			} else {
// 				// just add it
// 				eq[formula.Features[0]] = q
// 			}
// 		case GreaterRational:
// 			q := float64(f)
// 			// now update the greater entry to max
// 			// we have the condition that f1 > q
// 			// if we have already an entry stored we need the max of both.
// 			// for example f1 > 21 and f1 > 42 ==> f1 > 42
// 			if entry, has := greater[formula.Features[0]]; has {
// 				greater[formula.Features[0]] = math.Max(entry, q)
// 			} else {
// 				greater[formula.Features[0]] = q
// 			}
// 		case BinaryEqualsRational:
// 			// now add an entry that f1 and f2 must be equal
// 			f1, f2 := formula.Features[0], formula.Features[1]
// 			if inner, has := varEq[f1]; has {
// 				inner[f2] = struct{}{}
// 			} else {
// 				newMap := make(map[FeatureID]struct{}, numVars)
// 				newMap[f2] = struct{}{}
// 				varEq[f1] = newMap
// 			}
//
// 			// we want an undirected graph, so add the other direction as well
// 			if inner, has := varEq[f2]; has {
// 				inner[f1] = struct{}{}
// 			} else {
// 				newMap := make(map[FeatureID]struct{}, numVars)
// 				newMap[f1] = struct{}{}
// 				varEq[f2] = newMap
// 			}
// 		}
// 	}
// 	// after preprocessing we can check == and > (see comment above)
// 	// and also we can do some improvements on same variables
// 	for f, eqTo := range eq {
// 		// now check if there is an entry in >, if yes: if entry <= eqTo
// 		// return false, this will take care of the == problem and if entry < eqTo
// 		// we're already done because there is no possible solution
// 		if greaterThan, has := greater[f]; has {
// 			if eqTo <= greaterThan {
// 				return nil
// 			}
// 		}
// 	}
// 	// now we'll do another check:
// 	// all variables that are marked as equal must also confirm our extended
// 	// > test. For example imagine we have f1 = f2 = f3
// 	// and f3 has a constraint > 5 and we have f1 = 5. In this case we also
// 	// want to find this error. Thus we create all connected components
// 	// (imagine varEq as an undirected graph) and do the said test for all
// 	// variables in the connected components.
//
// 	cComponents := d.connectedComponents(varEq)
// 	for _, component := range cComponents {
// 		// the first thing we do is to check if there is an eq constraint on any
// 		// variable in the components and compute the maximum of all greater entries
// 		eqVal, eqFound := 0.0, false
// 		maxVal, maxFound := 0.0, false
// 		for fi, _ := range component {
// 			if next, has := eq[fi]; has {
// 				if eqFound {
// 					// if we already found a value it must be equal to this one,
// 					// otherwise there is no solution
// 					if next != eqVal {
// 						return nil
// 					}
// 					// if equal we have nothing to do, value is unchanged
// 				} else {
// 					eqVal = next
// 					eqFound = true
// 				}
// 			}
// 			// now check if there is an entry in the greater dictionary
// 			if next, has := greater[fi]; has {
// 				if maxFound {
// 					// no matter if we already found an entry, we can just update to max
// 					maxVal = math.Max(maxVal, next)
// 				} else {
// 					maxVal = next
// 					maxFound = true
// 				}
// 			}
// 			// if neither was found (>, eq) we don't have to worry
// 			if !eqFound && !maxFound {
// 				continue
// 			}
// 			// so now we have the eq val and the max val for the whole component
// 			// later on, when formulating the lp, for things that have an eq entry
// 			// we don't actually care about the > value because we already took care
// 			// of that constraint. so now we add to all elements in the component
// 			// the eq value and the max value (though this is as mentioned not
// 			// required).
// 			// but not all variables have eq set and we haven't checked yet if we
// 			// found a value, we'll do that all in the next loop
// 			for fi, _ := range component {
// 				// if eq value was found just set the eq value
// 				if eqFound {
// 					eq[fi] = eqVal
// 				}
// 				// if max value was found just update the max value entry
// 				if maxFound {
// 					greater[fi] = maxVal
// 				} else {
// 					// so we haven't found a max value, but there might be an
// 					// "evil" entry in greater that could harm us... so now look it up
// 					if next, has := greater[fi]; has {
// 						// now perform the same test as before
// 						// if there is no eq value however just ignore it
// 						if eqFound && eqVal <= next {
// 							return nil
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// 	lp := golp.NewLP(0, numVars)
// 	// now generate the actual LP
// 	// TODO
// 	fmt.Println(idMap)
// 	return lp
// }
//
// func (d RationalDomain) connectedComponents(varEq map[FeatureID]map[FeatureID]struct{}) []map[FeatureID]struct{} {
// 	res := make([]map[FeatureID]struct{}, 0, len(varEq))
// 	visited := make(map[FeatureID]struct{}, len(varEq))
// 	// visit each node in a for loop, run a search for that node and mark all
// 	// nodes reached as visited
// 	for node, _ := range varEq {
// 		if _, wasVisited := visited[node]; !wasVisited {
// 			res = append(res, d.bfs(varEq, node, visited))
// 		}
// 	}
// 	return res
// }
//
// func (d RationalDomain) bfs(varEq map[FeatureID]map[FeatureID]struct{}, node FeatureID, visited map[FeatureID]struct{}) map[FeatureID]struct{} {
// 	res := make(map[FeatureID]struct{}, len(varEq))
// 	// at least add node to result, not really desired I think but it doesn't hurt
// 	res[node] = struct{}{}
// 	queue := make([]FeatureID, 1, len(varEq))
// 	queue[0] = node
// 	for len(queue) > 0 {
// 		next := queue[0]
// 		queue = queue[1:]
// 		// if node was already visited there is nothing to do
// 		if _, wasVisited := visited[next]; wasVisited {
// 			// if node was already visited there is nothing to do
// 			continue
// 		}
// 		// mark node as visited
// 		visited[next] = struct{}{}
// 		// now expand the node
// 		for succ, _ := range varEq[next] {
// 			// if node was already visited there is nothing to do
// 			if _, wasVisited := visited[succ]; wasVisited {
// 				continue
// 			}
// 			// otherwise we found a new node in the component, so add to result
// 			res[succ] = struct{}{}
// 			// add to queue
// 			queue = append(queue, succ)
// 		}
// 	}
// 	return res
// }
//
// func (d RationalDomain) bla(f1, f2 FeatureID, eq map[FeatureID]float64, greater map[FeatureID]float64) bool {
// 	// eqOne, hasEqOne := eq[f1]
// 	// eqTwo, hasEqTwo := eq[f2]
// 	// greaterOne, hasGreaterOne := greater[f1]
// 	// greaterTwo, hasGreaterTwo := greater[f2]
// 	// switch {
// 	// case hasEqOne && hasEqTwo:
// 	// 	// if the values are not equal there is no solution
// 	// 	if eqOne != eqTwo {
// 	// 		return false
// 	// 	}
// 	// 	// if they are equal we check for entries in the other greater map and
// 	// 	// repeat the process we did before
// 	// 	if hasGreaterTwo {
// 	// 		// second one has an entry, do the <= check
// 	// 		if eqOne <= greater
// 	// 	}
// 	// case hasEqOne:
// 	// case hasEqTwo:
// 	// }
// 	return true
// }
//
// func (d RationalDomain) getIDMap(gamma ...*PredicateFormula) (map[FeatureID]int, int) {
// 	res := make(map[FeatureID]int)
// 	nextID := 0
// 	for _, formula := range gamma {
// 		for _, feature := range formula.Features {
// 			if _, has := res[feature]; !has {
// 				res[feature] = nextID
// 				nextID++
// 			}
// 		}
// 	}
// 	return res, nextID
// }
