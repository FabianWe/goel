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

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

//// Concepts ////

// TODO Add a better link tbox <-> components <-> domains, at the moment the
// solver get the domains

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

// NumBCD returns the number of elements in the basic concept description.
// That is the number of nominals, names, cd extensions + 1 (⊤).
func (c *ELBaseComponents) NumBCD() uint {
	return c.Nominals + c.CDExtensions + c.Names + 1
}

// TODO think this through, escpecially if one is 0!
func (c *ELBaseComponents) GetConcept(normalizedID uint) Concept {
	switch {
	default:
		// should never happen
		return nil
	case normalizedID == 0:
		return Bottom
	case normalizedID == 1:
		return Top
	case normalizedID < 2+c.Nominals:
		return NewNominalConcept(normalizedID - 2)
	case normalizedID < 2+c.Nominals+c.CDExtensions:
		return NewConcreteDomainExtension(normalizedID - 2 - c.Nominals)
	case normalizedID < 2+c.Nominals+c.CDExtensions+c.Names:
		return NewNamedConcept(normalizedID - 2 - c.Nominals - c.CDExtensions)
	}
}

func (c *ELBaseComponents) Write(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%d,%d,%d,%d", c.Nominals, c.CDExtensions, c.Names, c.Roles)
	return err
}

func ParseELBaseComponents(line string) (*ELBaseComponents, error) {
	split := strings.Split(line, ",")
	if len(split) != 4 {
		return nil, fmt.Errorf("Can't parse components from string, requires form \"int,int,int,int\", got %s", line)
	}
	values := make([]uint, 4)
	for i, str := range split {
		asInt, convErr := strconv.Atoi(str)
		if convErr != nil {
			return nil, convErr
		}
		values[i] = uint(asInt)
	}
	return NewELBaseComponents(values[0], values[1], values[2], values[3]), nil
}

// Concept is the interface for all Concept defintions.
// Concepts in EL++ are defined recursively, this is the general interface.
type Concept interface {
	// IsInBCD must return true for all concepts that are in the basic concept
	// description, these are: ⊤, all concept names, all concepts of the form {a}
	// and p(f1, ..., fk).
	IsInBCD() bool

	// NormalizedID is used to give each element of the basic concept description
	// and the bottom concept a unique id.
	// This might be useful if we want to store for example a set of elements from
	// the BCD.
	// The ids are defined in the following way:
	// The bottom concept has an id of 0
	// The top concept has an id of 1
	// All nominals have an id in 2...NumNoinals + 1
	// All CDExtensions have an id in NumNoinals + 2...NumNomials + NumCDExtensions + 1
	// All names haven an id in NumNomials + NumCDExtensions + 2....NumNomials + NumCDExtensions + NumNames + 1
	// This way we can easily add new names all the time, because the ids are at
	// the end of the representation and when we add a new name we don't have to
	// adjust all other ids in use (if this is ever required).
	//
	// So we can think of the representation in the wollowing way:
	// [⊥ ⊤ Individiaul1...IndividualN CDExtension1...CDExtensioN Name1...NameN]
	NormalizedID(c *ELBaseComponents) uint
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

func (bot BottomConcept) IsInBCD() bool {
	return false
}

func (bot BottomConcept) NormalizedID(c *ELBaseComponents) uint {
	return 0
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

func (top TopConcept) IsInBCD() bool {
	return true
}

func (top TopConcept) NormalizedID(c *ELBaseComponents) uint {
	return 1
}

// Top is a constant concept that represents to top concept ⊤.
var Top TopConcept = NewTopConcept()

// Bottom is a constant concept that represents the bottom concept ⊥.
var Bottom BottomConcept = NewBottomConcept()

// NamedConcept is a concept from the set of concept names, identified by an
// id.
// Each concept name A ∈ N_C is encoded as a unique integer with this type.
type NamedConcept uint

// NewNamedConcept returns a new NamedConcept with the given id.
func NewNamedConcept(i uint) NamedConcept {
	return NamedConcept(i)
}

func (name NamedConcept) String() string {
	return fmt.Sprintf("A(%d)", name)
}

func (name NamedConcept) IsInBCD() bool {
	return true
}

func (name NamedConcept) NormalizedID(c *ELBaseComponents) uint {
	return 2 + c.Nominals + c.CDExtensions + uint(name)
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

// NewNominalConcept returns a new NominalConcept with the given id.
func NewNominalConcept(i uint) NominalConcept {
	return NominalConcept(i)
}

func (nominal NominalConcept) String() string {
	return fmt.Sprintf("{a(%d)}", nominal)
}

func (nominal NominalConcept) IsInBCD() bool {
	return true
}

func (nominal NominalConcept) NormalizedID(c *ELBaseComponents) uint {
	return 2 + uint(nominal)
}

// Role is an EL++ role r ∈ N_R, identifiey by id.
// Each r ∈ N_R is encoded as a unique integer with this type.
type Role uint

// NewRole returns a new Rolen with the given id.
func NewRole(i uint) Role {
	return Role(i)
}

func (role Role) String() string {
	return fmt.Sprintf("r(%d)", role)
}

const (
	// NoRule is used to identify a role as not valid.
	NoRole Role = Role(^uint(0))
)

// ConcreteDomainExtension is a concrete domain extension of the form
// p(f1, ..., fk). All this information (predicate and function) has to be
// stored somewhere else, we only store the an id that identifies the concrete
// domain.
type ConcreteDomainExtension uint

// NewConcreteDomainExtension returns a new ConcreteDomainExtension with the given id.
func NewConcreteDomainExtension(i uint) ConcreteDomainExtension {
	return ConcreteDomainExtension(i)
}

func (cd ConcreteDomainExtension) String() string {
	return fmt.Sprintf("CD(%d)", cd)
}

func (cd ConcreteDomainExtension) IsInBCD() bool {
	return true
}

func (cd ConcreteDomainExtension) NormalizedID(c *ELBaseComponents) uint {
	return 2 + c.Nominals + uint(cd)
}

// Conjunction is a concept of the form C ⊓ D.
type Conjunction struct {
	// C, D are the parts of the conjuntion.
	C, D Concept
}

// NewConjunction returns a new conjunction given C and D.
func NewConjunction(c, d Concept) Conjunction {
	return Conjunction{C: c, D: d}
}

func NewMultiConjunction(concepts ...Concept) Concept {
	if len(concepts) == 0 {
		return Top
	}
	var res Concept
	res = concepts[0]
	for _, c := range concepts[1:] {
		res = NewConjunction(res, c)
	}
	return res
}

func (conjunction Conjunction) String() string {
	return fmt.Sprintf("(%v ⊓ %v)", conjunction.C, conjunction.D)
}

func (conjunction Conjunction) IsInBCD() bool {
	return false
}

func (conjunction Conjunction) NormalizedID(c *ELBaseComponents) uint {
	return ^uint(0)
}

// ExistentialConcept is a concept of the form ∃r.C.
type ExistentialConcept struct {
	R Role
	C Concept
}

// NewExistentialConcept returns a new existential concept of the form
// ∃r.C.
func NewExistentialConcept(r Role, c Concept) ExistentialConcept {
	return ExistentialConcept{R: r, C: c}
}

func (existential ExistentialConcept) String() string {
	return fmt.Sprintf("∃ %v.%v", existential.R, existential.C)
}

func (existential ExistentialConcept) IsInBCD() bool {
	return false
}

func (existential ExistentialConcept) NormalizedID(c *ELBaseComponents) uint {
	return ^uint(0)
}

// BCDOrFalse checks if the concept is either the bottom concept ⊥ or otherwise
// if it is in the BCD.
func BCDOrFalse(c Concept) bool {
	_, isFalse := c.(BottomConcept)
	return isFalse || c.IsInBCD()
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
	return fmt.Sprintf("%s ⊑ %s", strings.Join(strs, " o "), ri.RHS.String())
}

// TBox describes a TBox as a set of GCIs and RIs.
type TBox struct {
	Components *ELBaseComponents
	GCIs       []*GCIConstraint
	RIs        []*RoleInclusion
}

// NewTBox returns a new TBox.
func NewTBox(components *ELBaseComponents, gcis []*GCIConstraint, ris []*RoleInclusion) *TBox {
	return &TBox{Components: components, GCIs: gcis, RIs: ris}
}

//// Normalized TBox ////

// NormalizedCI is a normalized CI of the form C1 ⊓ C2 ⊑ D or C1 ⊑ D.
// For C1 ⊑ D C2 is set to nil.
// C1 and C2 must be in BC(T) and D must be in BC(T) or ⊥.
type NormalizedCI struct {
	// C1 and C2 form the LHS of the CI.
	C1, C2 Concept
	// D is the RHS of the CI
	D Concept
}

// NewNormalizedCI returns a new normalized CI of the form C1 ⊓ C2 ⊑ D.
// C1 and C2 must be in BC(T) and D must be in BC(T) or ⊥.
func NewNormalizedCI(c1, c2, d Concept) *NormalizedCI {
	return &NormalizedCI{C1: c1, C2: c2, D: d}
}

// NewNormalizedCISingle returns a new normalized CI of the form C1 ⊑ D.
func NewNormalizedCISingle(c1, d Concept) *NormalizedCI {
	return NewNormalizedCI(c1, nil, d)
}

func (ci *NormalizedCI) String() string {
	switch ci.C2 {
	case nil:
		return fmt.Sprintf("%v ⊑ %v", ci.C1, ci.D)
	default:
		return fmt.Sprintf("%v ⊓ %v ⊑ %v", ci.C1, ci.C2, ci.D)
	}
}

func writeTriple(w io.Writer, first, second, third int) error {
	_, err := fmt.Fprintf(w, "%d,%d,%d", first, second, third)
	return err
}

func parseTriple(line string) (first, second, third int, err error) {
	split := strings.Split(line, ",")
	if len(split) != 3 {
		err = fmt.Errorf("Can't parse components from string, requires form \"int,int,int\", got %s", line)
		return
	}
	values := make([]int, 3)
	for i, str := range split {
		asInt, convErr := strconv.Atoi(str)
		if convErr != nil {
			err = convErr
			return
		}
		values[i] = asInt
	}
	first, second, third = values[0], values[1], values[2]
	return
}

func (ci *NormalizedCI) Write(w io.Writer, c *ELBaseComponents) error {
	var first, second, third int
	first = int(ci.C1.NormalizedID(c))
	third = int(ci.D.NormalizedID(c))
	if ci.C2 == nil {
		second = -1
	} else {
		second = int(ci.C2.NormalizedID(c))
	}
	return writeTriple(w, first, second, third)
}

func ParseNormalizedCI(line string, c *ELBaseComponents) (*NormalizedCI, error) {
	first, second, third, err := parseTriple(line)
	if err != nil {
		return nil, err
	}
	var c1, c2, d Concept
	c1 = c.GetConcept(uint(first))
	d = c.GetConcept(uint(third))
	if second >= 0 {
		c2 = c.GetConcept(uint(second))
	}
	return NewNormalizedCI(c1, c2, d), nil
}

// NormalizedCIRightEx is a normalized CI where the existential quantifier is on
// the right-hand side, i.e. of the form C1 ⊑ ∃r.C2.
// C1 and C2 must be in BC(T) and D must be in BC(T) or ⊥.
type NormalizedCIRightEx struct {
	C1, C2 Concept
	R      Role
}

// NewNormalizedCIRightEx returns a new CI of the form C1 ⊑ ∃r.C2.
func NewNormalizedCIRightEx(c1 Concept, r Role, c2 Concept) *NormalizedCIRightEx {
	return &NormalizedCIRightEx{C1: c1, R: r, C2: c2}
}

func (ci *NormalizedCIRightEx) String() string {
	return fmt.Sprintf("%v ⊑ ∃ %s.%v", ci.C1, ci.R.String(), ci.C2)
}

func (ci *NormalizedCIRightEx) Write(w io.Writer, c *ELBaseComponents) error {
	return writeTriple(w, int(ci.C1.NormalizedID(c)),
		int(ci.R), int(ci.C2.NormalizedID(c)))
}

func ParseNormalizedCIRightEx(line string, c *ELBaseComponents) (*NormalizedCIRightEx, error) {
	first, second, third, err := parseTriple(line)
	if err != nil {
		return nil, err
	}
	return NewNormalizedCIRightEx(c.GetConcept(uint(first)),
		NewRole(uint(second)), c.GetConcept(uint(third))), nil
}

// NormalizedCILeftEx is a normalized CI where the existential quantifier is on
// the left-hand side, i.e. of the form ∃r.C1 ⊑ D.
// C1 must be in BC(T) and D must be in BC(T) or ⊥.
type NormalizedCILeftEx struct {
	C1, D Concept
	R     Role
}

// NewNormalizedCILeftEx returns a new CI of the form ∃r.C1 ⊑ D.
func NewNormalizedCILeftEx(r Role, c1, d Concept) *NormalizedCILeftEx {
	return &NormalizedCILeftEx{C1: c1, D: d, R: r}
}

func (ci *NormalizedCILeftEx) String() string {
	return fmt.Sprintf("∃ %s.%v ⊑ %v", ci.R.String(), ci.C1, ci.D)
}

func (ci *NormalizedCILeftEx) Write(w io.Writer, c *ELBaseComponents) error {
	return writeTriple(w, int(ci.R), int(ci.C1.NormalizedID(c)), int(ci.D.NormalizedID(c)))
}

func ParseNormalizedCILeftEx(line string, c *ELBaseComponents) (*NormalizedCILeftEx, error) {
	first, second, third, err := parseTriple(line)
	if err != nil {
		return nil, err
	}
	return NewNormalizedCILeftEx(NewRole(uint(first)), c.GetConcept(uint(second)), c.GetConcept(uint(third))), nil
}

// NormalizedRI is a normalized RI of the form r1 o r2 ⊑ s or r1 ⊑ s.
// For the second form R2 is set to NoRole.
type NormalizedRI struct {
	R1, R2, S Role
}

// NewNormalizedRI returns a new normalized RI of the form r1 o r2 ⊑ s.
func NewNormalizedRI(r1, r2, s Role) *NormalizedRI {
	return &NormalizedRI{R1: r1, R2: r2, S: s}
}

// NewNormalizedRISingle returns a new normalized RI of the form r1 ⊑ s.
func NewNormalizedRISingle(r1, s Role) *NormalizedRI {
	return NewNormalizedRI(r1, NoRole, s)
}

func (ri *NormalizedRI) String() string {
	if ri.R2 == NoRole {
		return fmt.Sprintf("%v ⊑ %v", ri.R1, ri.S)
	} else {
		return fmt.Sprintf("%v o %v ⊑ %v", ri.R1, ri.R2, ri.S)
	}
}

// TODO should be called ri... copied something wrong it seems...
func (ci *NormalizedRI) Write(w io.Writer) error {
	var first, second, third int
	first = int(ci.R1)
	third = int(ci.S)
	if ci.R2 == NoRole {
		second = -1
	} else {
		second = int(ci.R2)
	}
	return writeTriple(w, first, second, third)
}

func ParseNormalizedRI(line string) (*NormalizedRI, error) {
	first, second, third, err := parseTriple(line)
	if err != nil {
		return nil, err
	}
	var r1, r2, s Role
	r1 = NewRole(uint(first))
	s = NewRole(uint(third))
	if second < 0 {
		r2 = NoRole
	} else {
		r2 = NewRole(uint(second))
	}
	return NewNormalizedRI(r1, r2, s), nil
}

// NormalizedTBox is a TBox containing only normalized CIs.
type NormalizedTBox struct {
	Components *ELBaseComponents
	CIs        []*NormalizedCI
	CILeft     []*NormalizedCILeftEx
	CIRight    []*NormalizedCIRightEx
	RIs        []*NormalizedRI
}

func (tbox *NormalizedTBox) Write(w io.Writer) error {
	// first write components + newline
	if err := tbox.Components.Write(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return err
	}
	// now write length of CI, CILeft, CIRight and RI
	if _, err := fmt.Fprintf(w, "%d,%d,%d,%d\n", len(tbox.CIs), len(tbox.CILeft),
		len(tbox.CIRight), len(tbox.RIs)); err != nil {
		return err
	}
	// now write everything
	for _, ci := range tbox.CIs {
		if err := ci.Write(w, tbox.Components); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
	}
	for _, ci := range tbox.CILeft {
		if err := ci.Write(w, tbox.Components); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
	}
	for _, ci := range tbox.CIRight {
		if err := ci.Write(w, tbox.Components); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
	}
	for _, ri := range tbox.RIs {
		if err := ri.Write(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, ""); err != nil {
			return err
		}
	}
	return nil
}

func scannerAdvanceLine(s *bufio.Scanner) (string, error) {
	if !s.Scan() {
		if err := s.Err(); err != nil {
			return "", err
		} else {
			// reached EOF
			return "", errors.New("Too few input lines while reading Normalized TBox")
		}
	}
	return s.Text(), nil
}

func initNormalizedTbox(line string) (*NormalizedTBox, error) {
	split := strings.Split(line, ",")
	if len(split) != 4 {
		return nil, errors.New("Normalized TBox size line must contain for integers")
	}
	// parse integers
	values := make([]int, 4)
	for i, str := range split {
		// try to parse as int
		asInt, convErr := strconv.Atoi(str)
		if convErr != nil {
			return nil, convErr
		}
		values[i] = asInt
	}
	// now initialize the slices
	cis := make([]*NormalizedCI, values[0])
	ciLeft := make([]*NormalizedCILeftEx, values[1])
	ciRight := make([]*NormalizedCIRightEx, values[2])
	ris := make([]*NormalizedRI, values[3])
	// CIs        []*NormalizedCI
	// CILeft     []*NormalizedCILeftEx
	// CIRight    []*NormalizedCIRightEx
	// RIs        []*NormalizedRI
	return &NormalizedTBox{
		Components: nil,
		CIs:        cis,
		CILeft:     ciLeft,
		CIRight:    ciRight,
		RIs:        ris,
	}, nil
}

func ParseNormalizedTBox(r io.Reader) (*NormalizedTBox, error) {
	// create scanner to read input
	s := bufio.NewScanner(r)
	// first parse the components line
	var components *ELBaseComponents
	if line, err := scannerAdvanceLine(s); err != nil {
		return nil, err
	} else {
		// try to parse the components
		var compErr error
		components, compErr = ParseELBaseComponents(line)
		if compErr != nil {
			return nil, compErr
		}
	}
	// now try to parse the length of the tbox and create the slices
	var res *NormalizedTBox
	if line, initLineErr := scannerAdvanceLine(s); initLineErr != nil {
		return nil, initLineErr
	} else {
		var initErr error
		res, initErr = initNormalizedTbox(line)
		if initErr != nil {
			return nil, initErr
		}
	}
	res.Components = components
	// now parse the actual content
	for i := 0; i < len(res.CIs); i++ {
		if line, lineErr := scannerAdvanceLine(s); lineErr != nil {
			return nil, lineErr
		} else {
			if ci, ciParseErr := ParseNormalizedCI(line, components); ciParseErr != nil {
				return nil, ciParseErr
			} else {
				res.CIs[i] = ci
			}
		}
	}
	for i := 0; i < len(res.CILeft); i++ {
		if line, lineErr := scannerAdvanceLine(s); lineErr != nil {
			return nil, lineErr
		} else {
			if ci, ciParseErr := ParseNormalizedCILeftEx(line, components); ciParseErr != nil {
				return nil, ciParseErr
			} else {
				res.CILeft[i] = ci
			}
		}
	}
	for i := 0; i < len(res.CIRight); i++ {
		if line, lineErr := scannerAdvanceLine(s); lineErr != nil {
			return nil, lineErr
		} else {
			if ci, ciParseErr := ParseNormalizedCIRightEx(line, components); ciParseErr != nil {
				return nil, ciParseErr
			} else {
				res.CIRight[i] = ci
			}
		}
	}
	for i := 0; i < len(res.RIs); i++ {
		if line, lineErr := scannerAdvanceLine(s); lineErr != nil {
			return nil, lineErr
		} else {
			if ri, riParseErr := ParseNormalizedRI(line); riParseErr != nil {
				return nil, riParseErr
			} else {
				res.RIs[i] = ri
			}
		}
	}
	return res, nil
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
	return fmt.Sprintf("%s(%v, %v)", ra.R.String(), ra.A, ra.B)
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
