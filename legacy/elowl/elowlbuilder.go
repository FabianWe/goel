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

package elowl

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"gkigit.informatik.uni-freiburg.de/wenzelmf/2016-fwenzelmann-el-subsumption/elconc"
)

const RDFType = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
const RDFClass = "http://www.w3.org/2002/07/owl#Class"
const RDFSubclass = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
const RDFRestriction = "http://www.w3.org/2002/07/owl#Restriction"
const RDFOnProperty = "http://www.w3.org/2002/07/owl#onProperty"
const RDFSomeValuesFrom = "http://www.w3.org/2002/07/owl#someValuesFrom"
const RDFNil = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
const RDFList = "http://www.w3.org/1999/02/22-rdf-syntax-ns#List"
const RDFFirst = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
const RDFRest = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
const RDFComment = "http://www.w3.org/2000/01/rdf-schema#comment"
const RDFLabel = "http://www.w3.org/2000/01/rdf-schema#label"
const RDFVersionInfo = "http://www.w3.org/2002/07/owl#versionInfo"
const RDFObjectProperty = "http://www.w3.org/2002/07/owl#ObjectProperty"
const RDFSubPropertyOf = "http://www.w3.org/2000/01/rdf-schema#subPropertyOf"
const RDFDomain = "http://www.w3.org/2000/01/rdf-schema#domain"
const RDFRange = "http://www.w3.org/2000/01/rdf-schema#range"
const RDFInverseOf = "http://www.w3.org/2002/07/owl#inverseOf"
const RDFTransitiveProperty = "http://www.w3.org/2002/07/owl/#TransitiveProperty"
const RDFDatatypeProperty = "http://www.w3.org/2002/07/owl/#DatatypeProperty"

const blank_template = "_:genid:%v"

func IsBlankNode(s string) bool {
	// simplified, but should do
	return strings.HasPrefix(s, "_:genid:")
}

type RDFObjectType interface{}

func EqualsRDFObject(object, other RDFObjectType) bool {
	switch t1 := object.(type) {
	default:
		log.Fatalf("Invalid type for RDFObject: %v", reflect.TypeOf(object))
		return false
	case int:
		if val, ok := other.(int); ok {
			return t1 == val
		}
	case float64:
		if val, ok := other.(float64); ok {
			return t1 == val
		}
	case int32:
		if val, ok := other.(int32); ok {
			return t1 == val
		}
	case int64:
		if val, ok := other.(int64); ok {
			return t1 == val
		}
	case float32:
		if val, ok := other.(float32); ok {
			return t1 == val
		}
	case bool:
		if val, ok := other.(bool); ok {
			return t1 == val
		}
	case string:
		if val, ok := other.(string); ok {
			return t1 == val
		}
	}
	return false
}

func GetObjectString(o RDFObjectType) (string, error) {
	val, ok := o.(string)
	if !ok {
		return "", errors.New("RDFObject is not of type string")
	}
	return val, nil
}

type RDFTriple struct {
	Subject, Predicate string
	Object             RDFObjectType
}

func NewRDFTriple(subject, predicate string, object RDFObjectType) *RDFTriple {
	return &RDFTriple{Subject: subject,
		Predicate: predicate,
		Object:    object}
}

func (triple *RDFTriple) String() string {
	return fmt.Sprintf("(%s, %s, %s)", triple.Subject,
		triple.Predicate,
		triple.Object)
}

type OWLBuilder interface {
	HandleTriple(t *RDFTriple) error
	GetBlankNode() string
}

type TripleQueryHandler interface {
	AnswerQuery(subject, predicate *string, object RDFObjectType,
		f func(t *RDFTriple) error) error
}

//// DEFAULT BUILDER ////

type TripleMap map[string]map[string][]RDFObjectType

func NewTripleMap() TripleMap {
	return make(map[string]map[string][]RDFObjectType)
}

func (tm TripleMap) AddElement(key1, key2 string, value RDFObjectType) {
	inner, existsInner := tm[key1]
	if !existsInner {
		inner = make(map[string][]RDFObjectType)
		tm[key1] = inner
	}
	values := inner[key2]
	inner[key2] = append(values, value)
}

type DefaultOWLBuilder struct {
	NextBlankID  uint
	SubjectMap   TripleMap
	PredicateMap TripleMap
}

func NewDefaultOWLBuilder() *DefaultOWLBuilder {
	return &DefaultOWLBuilder{NextBlankID: 0,
		SubjectMap:   NewTripleMap(),
		PredicateMap: NewTripleMap()}
}

func (handler *DefaultOWLBuilder) GetBlankNode() string {
	res := fmt.Sprintf(blank_template, handler.NextBlankID)
	handler.NextBlankID++
	return res
}

func (handler *DefaultOWLBuilder) HandleTriple(t *RDFTriple) error {
	// add to both maps
	handler.SubjectMap.AddElement(t.Subject, t.Predicate, t.Object)
	handler.PredicateMap.AddElement(t.Predicate, t.Subject, t.Object)
	return nil
}

func (handler *DefaultOWLBuilder) AnswerQuery(subject, predicate *string, object RDFObjectType,
	f func(t *RDFTriple) error) error {
	dict := handler.SubjectMap
	first := subject
	second := predicate
	tFunc := func(firstElem, secondElem string, object RDFObjectType) *RDFTriple {
		return NewRDFTriple(firstElem, secondElem, object)
	}
	if subject == nil && predicate != nil {
		dict = handler.PredicateMap
		first = predicate
		second = subject
		tFunc = func(firstElem, secondElem string, object RDFObjectType) *RDFTriple {
			return NewRDFTriple(secondElem, firstElem, object)
		}
	}
	return handler.iterateDict(dict, first, second, object, f, tFunc)
}

// Iterate over the dictionary with first as the first key, if nil use all keys.
// In this use only those that satisfy second.
func (handler *DefaultOWLBuilder) iterateDict(dict TripleMap, first, second *string, o RDFObjectType,
	f func(t *RDFTriple) error, makeTripleFunc func(firstElem, secondElem string, object RDFObjectType) *RDFTriple) error {
	if first == nil {
		// iterate over all entries
		for key, d := range dict {
			if err := handler.iterateInnerDict(d, key, second, o, f, makeTripleFunc); err != nil {
				return err
			}
		}
	} else {
		// only use the given entry
		if d, ok := dict[*first]; ok {
			if err := handler.iterateInnerDict(d, *first, second, o, f, makeTripleFunc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (handler *DefaultOWLBuilder) iterateInnerDict(dict map[string][]RDFObjectType, first string, second *string, o RDFObjectType,
	f func(t *RDFTriple) error, makeTripleFunc func(firstElem, secondElem string, object RDFObjectType) *RDFTriple) error {
	// first check if second is nil, in this case iterate over all elements in the dict
	if second == nil {
		for innerKey, objects := range dict {
			// iterate over entries
			if err := handler.iterateObjectList(objects, first, innerKey, o, f, makeTripleFunc); err != nil {
				return err
			}
		}
	} else {
		// only check objects with the given key
		if objects, ok := dict[*second]; ok {
			if err := handler.iterateObjectList(objects, first, *second, o, f, makeTripleFunc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (handler *DefaultOWLBuilder) iterateObjectList(list []RDFObjectType, first, second string, o RDFObjectType,
	f func(t *RDFTriple) error, makeTripleFunc func(firstElem, secondElem string, object RDFObjectType) *RDFTriple) error {
	if o == nil {
		// iterate over all objects
		for _, object := range list {
			if err := f(makeTripleFunc(first, second, object)); err != nil {
				return err
			}
		}
	} else {
		// iterate only over those objects that are equal to o
		for _, object := range list {
			if EqualsRDFObject(object, o) {
				if err := f(makeTripleFunc(first, second, object)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type TBoxConverter struct {
	Classes, BlankClasses []string
	ClassID               map[string]int
	Relations             []*elconc.BinaryObjectRelation
	RelationNames         []string
	RelationID            map[string]int
	SubProperties         []*elconc.SubProp
}

func NewTBoxConverter() *TBoxConverter {
	return &TBoxConverter{}
}

func (converter *TBoxConverter) ConvertToTBox(handler TripleQueryHandler) error {
	converter.init()
	converter.getClasses(handler)
	if err := converter.getObjectProperties(handler); err != nil {
		return err
	}
	if err := converter.getSubproperties(handler); err != nil {
		return err
	}
	if err := converter.inverseProperties(handler); err != nil {
		return err
	}
	return nil
}

func (converter *TBoxConverter) init() {
	converter.Classes = nil
	converter.BlankClasses = nil
	converter.ClassID = make(map[string]int)
	converter.Relations = nil
	converter.RelationNames = nil
	converter.RelationID = make(map[string]int)
	converter.SubProperties = nil
}

func (converter *TBoxConverter) getClasses(handler TripleQueryHandler) {
	classFunc := func(t *RDFTriple) error {
		curClass := t.Subject
		if IsBlankNode(curClass) {
			converter.BlankClasses = append(converter.BlankClasses, curClass)
		} else {
			converter.ClassID[curClass] = len(converter.Classes)
			converter.Classes = append(converter.Classes, curClass)
		}
		return nil
	}
	// bit ugly, but ok
	t := RDFType
	handler.AnswerQuery(nil, &t, RDFClass, classFunc)
}

func (converter *TBoxConverter) getObjectProperties(handler TripleQueryHandler) error {
	// we need to iterate over all http://www.w3.org/2002/07/owl#ObjectProperty elements
	typeS := RDFType
	handleFunc := func(t *RDFTriple) error {
		// get the info
		relation, handleErr := converter.getOPInfo(handler, t.Subject)
		if handleErr != nil {
			return handleErr
		}
		converter.RelationID[t.Subject] = len(converter.Relations)
		converter.Relations = append(converter.Relations, relation)
		converter.RelationNames = append(converter.RelationNames, t.Subject)
		return nil
	}
	if err := handler.AnswerQuery(nil, &typeS, RDFObjectProperty, handleFunc); err != nil {
		return err
	}
	return nil
}

func (converter *TBoxConverter) getOPInfo(handler TripleQueryHandler, name string) (*elconc.BinaryObjectRelation, error) {
	var curRange, curDomain string
	// handler funcs that retrieves range and domain
	rangeFunc := func(t *RDFTriple) error {
		if curRange != "" {
			return fmt.Errorf("Got multiple range entries for \"%s\": Got \"%s\" and \"%s\"", name, curRange, t.Object)
		}
		objStr, err := GetObjectString(t.Object)
		if err != nil {
			return err
		}
		curRange = objStr
		return nil
	}
	domainFunc := func(t *RDFTriple) error {
		if curDomain != "" {
			return fmt.Errorf("Got multiple domain entries for \"%s\": Got \"%s\" and \"%s\"", name, curDomain, t.Object)
		}
		objStr, err := GetObjectString(t.Object)
		if err != nil {
			return err
		}
		curDomain = objStr
		return nil
	}
	ch := make(chan error, 2)
	// start both functions
	go func() {
		rangeS := RDFRange
		ch <- handler.AnswerQuery(&name, &rangeS, nil, rangeFunc)
	}()
	go func() {
		domainS := RDFDomain
		ch <- handler.AnswerQuery(&name, &domainS, nil, domainFunc)
	}()
	var err error
	for i := 0; i < 2; i++ {
		if nextErr := <-ch; nextErr != nil {
			if err == nil {
				err = nextErr
			}
		}
	}
	if err != nil {
		return nil, err
	}
	rangeID, hasRange := converter.ClassID[curRange]
	domainID, hasDomain := converter.ClassID[curDomain]
	// if both range and domain are missing, check if this property is a subproperty
	// in this case we simply use range and domain from them
	if !hasRange {
		// TODO I don't understand it but it happens in our benchmarks
		// (publicationDate)
		// this is never used later on, so what?
		rangeID = -1
		// return nil, fmt.Errorf("Unkown class as object range for \"%s\": \"%s\"", name, curRange)
	}
	if !hasDomain {
		// TODO I don't understand it but it happens in our benchmarks
		// (publicationDate)
		// this is never used later on, so what?
		domainID = -1
		// return nil, fmt.Errorf("Unkown class as object domain for \"%s\": \"%s\"", name, curDomain)
	}
	return elconc.NewBinaryObjectRelation(rangeID, domainID), nil
}

func (converter *TBoxConverter) getSubproperties(handler TripleQueryHandler) error {
	handleFunc := func(t *RDFTriple) error {
		// first get the id of the property the is the subproperty (lhs) of another
		// property (rhs)
		lhsID, hasLHS := converter.RelationID[t.Subject]
		if !hasLHS {
			return fmt.Errorf("Unkown property name \"%s\" while parsing subproperties", t.Subject)
		}
		if err := converter.resolveProperty(handler, lhsID); err != nil {
			return err
		}
		// no get rhs as well
		rhsName, nameErr := GetObjectString(t.Object)
		if nameErr != nil {
			return nameErr
		}
		rhsID, hasRHS := converter.RelationID[rhsName]
		if !hasRHS {
			return fmt.Errorf("Unkown property name \"%s\" while parsing subproperties", rhsName)
		}
		converter.SubProperties = append(converter.SubProperties, elconc.NewSubProp(lhsID, rhsID))
		return nil
	}
	subPropS := RDFSubPropertyOf
	return handler.AnswerQuery(nil, &subPropS, nil, handleFunc)
}

// Resolve the domain and range of a SubPropertyOf element by checking it's
// parents.
// TODO Maybe this should check for some conflicts, but I don't really care
func (converter *TBoxConverter) resolveProperty(handler TripleQueryHandler, propID int) error {
	rel := converter.Relations[propID]
	// try to get the values from a "parent". We need to do this recursively
	// we keep a list of already visited relations in order to avoid loops
	visited := make(map[int]struct{})
	waiting := make([]int, 1)
	waiting[0] = propID
	for len(waiting) > 0 {
		n := len(waiting)
		nextID := waiting[n-1]
		waiting = waiting[:n-1]
		nextRel := converter.Relations[nextID]
		if rel.Domain < 0 && nextRel.Domain > -1 {
			rel.Domain = nextRel.Domain
		}
		if rel.Range < 0 && nextRel.Range > -1 {
			rel.Range = nextRel.Range
		}
		if rel.Domain > -1 && rel.Range > -1 {
			return nil
		}
		// mark as visited
		visited[nextID] = struct{}{}
		// if we're not done yet we need to add all "parents"
		handleFunc := func(t *RDFTriple) error {
			// get the RHS (the "parent") and add its id to waiting (if not visited)
			// before
			rhsName, nameErr := GetObjectString(t.Object)
			if nameErr != nil {
				return nameErr
			}
			rhsID, hasRHS := converter.RelationID[rhsName]
			if !hasRHS {
				return fmt.Errorf("Unkown property name \"%s\" while parsing subproperties", rhsName)
			}
			// check if it is already in visited
			if _, alreadyVisited := visited[rhsID]; !alreadyVisited {
				// if not visited: add to waiting
				waiting = append(waiting, rhsID)
			}
			return nil
		}
		// apply handle function
		subPropS := RDFSubPropertyOf
		if err := handler.AnswerQuery(&converter.RelationNames[nextID], &subPropS, nil, handleFunc); err != nil {
			return err
		}
	}
	// return fmt.Errorf("Can't resolve domain and range for property \"%s\"", converter.RelationNames[propID])
	// TODO this is what we usually would do, but the benchmark seems to be not
	// correct here...
	// so we simply ignore this
	// So we need to take inverseOf elements... we'll do that next
	return nil
}

func (converter *TBoxConverter) inverseProperties(handler TripleQueryHandler) error {
	fmt.Println("JO")
	return nil
}

//////////// OLD IMPLEMENTATION, JUST USE THAT AS A REFERENCE NOW

// type valueRestriction struct {
// 	PropertyResource string
// 	ValuesFrom       string
// }
//
// type rdfListEntry struct {
// 	Item, Rest string
// }
//
// type OldOWLBuilder struct {
// 	nextBlankID       int
// 	Comments          URIDict
// 	Labels            URIDict
// 	Subclasses        map[string][]string
// 	Types             map[string][]string
// 	ValueRestrictions map[string]*valueRestriction
// 	Lists             map[string]*rdfListEntry
// 	SubProperties     map[string][]string
// 	ObjectDomains     map[string][]string
// 	ObjectRanges      map[string][]string
// 	InverseOf         URIDict
// }
//
// func (this *OldOWLBuilder) addSubclass(subject string, object string) {
// 	subclasses, _ := this.Subclasses[subject]
// 	subclasses = append(subclasses, object)
// 	this.Subclasses[subject] = subclasses
// }
//
// func NewOWLBuilder() *OldOWLBuilder {
// 	comments := make(map[string]string)
// 	labels := make(map[string]string)
// 	subclasses := make(map[string][]string)
// 	types := make(map[string][]string)
// 	valueRestrictions := make(map[string]*valueRestriction)
// 	lists := make(map[string]*rdfListEntry)
// 	subProperties := make(map[string][]string)
// 	objectDomains := make(map[string][]string)
// 	objectRanges := make(map[string][]string)
// 	inverseOf := make(map[string]string)
// 	return &OldOWLBuilder{Comments: comments,
// 		Labels:            labels,
// 		Subclasses:        subclasses,
// 		Types:             types,
// 		ValueRestrictions: valueRestrictions,
// 		Lists:             lists,
// 		SubProperties:     subProperties,
// 		ObjectDomains:     objectDomains,
// 		ObjectRanges:      objectRanges,
// 		InverseOf:         inverseOf,
// 		nextBlankID:       0}
// }
//
// func (this *OldOWLBuilder) GetBlankNode() string {
// 	res := fmt.Sprintf(blank_template, this.nextBlankID)
// 	this.nextBlankID++
// 	return res
// }
//
// func (this *OldOWLBuilder) ProcessTriple(t *RDFTriple) error {
// 	switch t.Predicate {
// 	default:
// 		log.Printf("Unkown predicate in triple: %s\n", t.Predicate)
// 	case RDFComment:
// 		this.Comments.SetElement(t.Subject, t.Object)
// 	case RDFLabel:
// 		this.Labels.SetElement(t.Predicate, t.Object)
// 	case RDFType:
// 		subjectStr := t.Subject
// 		s, ok := this.Types[subjectStr]
// 		if !ok {
// 			s = nil
// 		}
// 		this.Types[subjectStr] = append(s, t.Object)
// 	case RDFVersionInfo:
// 		// ingore right now...
// 	case RDFSubclass:
// 		this.addSubclass(t.Subject, t.Object)
// 	case RDFOnProperty:
// 		if err := this.handleOnProperty(t); err != nil {
// 			return err
// 		}
// 	case RDFSomeValuesFrom:
// 		if err := this.handleSomeValuesFrom(t); err != nil {
// 			return err
// 		}
// 	case RDFFirst:
// 		if err := this.handleListFirst(t); err != nil {
// 			return err
// 		}
// 	case RDFRest:
// 		if err := this.handleListRest(t); err != nil {
// 			return err
// 		}
// 	case RDFSubPropertyOf:
// 		subjectStr := t.Subject
// 		s, ok := this.SubProperties[subjectStr]
// 		if !ok {
// 			s = nil
// 		}
// 		this.SubProperties[subjectStr] = append(s, t.Object)
// 	case RDFDomain:
// 		subjectStr := t.Subject
// 		s, ok := this.ObjectDomains[subjectStr]
// 		if !ok {
// 			s = nil
// 		}
// 		this.ObjectDomains[subjectStr] = append(s, t.Object)
// 	case RDFRange:
// 		subjecStr := t.Subject
// 		s, ok := this.ObjectRanges[subjecStr]
// 		if !ok {
// 			s = nil
// 		}
// 		this.ObjectRanges[subjecStr] = append(s, t.Object)
// 	case RDFInverseOf:
// 		this.InverseOf.SetElement(t.Subject, t.Object)
// 	}
// 	return nil
// }
//
// func (builder *OldOWLBuilder) HasOWLType(itemName string, names ...string) bool {
// 	types, found := builder.Types[itemName]
// 	if !found {
// 		return false
// 	}
// 	for _, name := range names {
// 		for _, actualName := range types {
// 			if name == actualName {
// 				return true
// 			}
// 		}
// 	}
// 	return false
// }
//
// func (builder *OldOWLBuilder) IsList(itemName string) bool {
// 	return itemName == RDFNil || builder.HasOWLType(itemName, RDFList)
// }
//
// func (this *OldOWLBuilder) getValueRestriction(uri string) (*valueRestriction, error) {
// 	// we have to make sure that the subject has a type owl:Restriction
// 	if !this.HasOWLType(uri, RDFRestriction) {
// 		return nil, fmt.Errorf("Element %s has no type set to owl:Restriction", uri)
// 	}
// 	vr, found := this.ValueRestrictions[uri]
// 	if !found {
// 		vr = &valueRestriction{PropertyResource: "", ValuesFrom: ""}
// 		this.ValueRestrictions[uri] = vr
// 	}
// 	return vr, nil
// }
//
// func (builder *OldOWLBuilder) getListEntry(uri string) (*rdfListEntry, error) {
// 	// ensure that it has type rdf:List
// 	if !builder.IsList(uri) {
// 		return nil, fmt.Errorf("Element %s is not of type rdf:List", uri)
// 	}
// 	le, found := builder.Lists[uri]
// 	if !found {
// 		le = &rdfListEntry{Item: "", Rest: ""}
// 		builder.Lists[uri] = le
// 	}
// 	return le, nil
// }
//
// func (this *OldOWLBuilder) handleOnProperty(t *RDFTriple) error {
// 	vr, err := this.getValueRestriction(t.Subject)
// 	if err != nil {
// 		return err
// 	}
// 	if vr.PropertyResource != "" {
// 		log.Printf("WARNING: Already got a property for %s\n", t.Subject)
// 	}
// 	vr.PropertyResource = t.Object
// 	return nil
// }
//
// func (this *OldOWLBuilder) handleSomeValuesFrom(t *RDFTriple) error {
// 	vr, err := this.getValueRestriction(t.Subject)
// 	if err != nil {
// 		return err
// 	}
// 	if vr.ValuesFrom != "" {
// 		log.Printf("WARNING: Already got a value restriction for %s\n", t.Subject)
// 	}
// 	vr.ValuesFrom = t.Object
// 	return nil
// }
//
// func (builder *OldOWLBuilder) handleListFirst(t *RDFTriple) error {
// 	le, err := builder.getListEntry(t.Subject)
// 	if err != nil {
// 		return err
// 	}
// 	if le.Item != "" {
// 		log.Printf("WARNING: Already got an item for list %s", t.Subject)
// 	}
// 	le.Item = t.Object
// 	return nil
// }
//
// func (builder *OldOWLBuilder) handleListRest(t *RDFTriple) error {
// 	le, err := builder.getListEntry(t.Subject)
// 	if err != nil {
// 		return err
// 	}
// 	if le.Rest != "" {
// 		log.Printf("WARNING: Already got rest for list %s", t.Subject)
// 	}
// 	le.Rest = t.Object
// 	return nil
// }
