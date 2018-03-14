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
	"io"
	"log"
	"strings"
	"sync"
)

// An interface for everything that parses triples from a given reader and
// adds those triples to the builder for further processing.
type TurtleParser interface {
	Parse(r io.Reader, defaultBase string, builder OWLBuilder) error
}

// Abstract Syntax Tree (AST) //

// A type for the nonternimals as defined in the Turtle grammar. Nearly
// the same as in the formal specification, though we don't support everything
// just yet.
type TurtleNonterminal int

const (
	TurtleDoc TurtleNonterminal = iota
	Statement
	NonterminalPrefixID
	NonterminalBase
	Directive
	Subject
	IRI
	BlankNode
	Collection
	PrefixedName
	Triples
	Object
	BlankNodePropertyList
	Literal
	String
	ObjectList
	PredicateObjectList
	Verb
	Predicate
	RDFLiteral
)

// Human readable version.
func (n TurtleNonterminal) String() string {
	switch n {
	case TurtleDoc:
		return "TurtleDoc"
	case Statement:
		return "Statement"
	case NonterminalPrefixID:
		return "NonterminalPrefixID"
	case NonterminalBase:
		return "NonterminalBase"
	case Directive:
		return "Directive"
	case Subject:
		return "Subject"
	case IRI:
		return "IRI"
	case BlankNode:
		return "BlankNode"
	case Collection:
		return "Collection"
	case PrefixedName:
		return "PrefixedName"
	case Triples:
		return "Triples"
	case Object:
		return "Object"
	case BlankNodePropertyList:
		return "BlankNodePropertyList"
	case Literal:
		return "Literal"
	case String:
		return "String"
	case ObjectList:
		return "ObjectList"
	case PredicateObjectList:
		return "PredicateObjectList"
	case Verb:
		return "Verb"
	case Predicate:
		return "Predicate"
	case RDFLiteral:
		return "RDFLiteral"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", n)
	}
}

// A special error that is used to specifiy that we expected to read a token
// but no further token was found.
// This way you can check if there was an error because a rule didn't match
// or simply the stream was empty.
var ErrNoToken = errors.New("Expected token, but non was found.")

type converterState struct {
	base         string
	curSubject   string
	curPredicate string
}

func newConverterState() *converterState {
	return &converterState{
		base:         "",
		curSubject:   "",
		curPredicate: ""}
}

func (state *converterState) copy() *converterState {
	return &converterState{base: state.base,
		curSubject:   state.curSubject,
		curPredicate: state.curPredicate}
}

func (state *converterState) String() string {
	return fmt.Sprintf("Base: \"%s\" Subject: \"%s\" Predicate: \"%s\"",
		state.base, state.curSubject, state.curPredicate)
}

type ASTConverter interface {
	Convert(ast *TurtleAST, defaultBase string, builder OWLBuilder) error
}

// Used passing around dictionary entries while parsing @prefix
type prefixEntry struct {
	key, value string
}

func newPrefixEntry(key, value string) *prefixEntry {
	return &prefixEntry{key: key, value: value}
}

type DefaultASTConverter struct {
	numStatementWorkers int
	base                string
	namespaces          map[string]string
	bnodeLabels         map[string]string
	ast                 *TurtleAST
	builder             OWLBuilder
	// error handling
	firstErr error
	errChan  chan error
	errDone  chan bool
	// reporting triples
	triplesChan chan *RDFTriple
	triplesDone chan bool
	// parsing round one
	prefixChan     chan *prefixEntry
	prefixDone     chan bool
	blockBlankNode chan bool
}

func NewDefaultASTConverter(numStatementWorkers int) *DefaultASTConverter {
	if numStatementWorkers <= 0 {
		numStatementWorkers = 1
	}
	return &DefaultASTConverter{numStatementWorkers: numStatementWorkers,
		base:           "",
		namespaces:     nil,
		bnodeLabels:    nil,
		ast:            nil,
		builder:        nil,
		firstErr:       nil,
		errChan:        nil,
		errDone:        nil,
		triplesChan:    nil,
		triplesDone:    nil,
		prefixChan:     nil,
		prefixDone:     nil,
		blockBlankNode: nil}
}

// Return a function that reads from the error channel and sets the
// firstErr variable if it is nil to the first next nil error it gets.
// This is used s.t. multiple functions can report errors to one converter
// object.
// The method is usually started by the convert method and the channel is closed
// once we're done parsing. However you should wait for the errDone channel
// becuase this method reports to this channel when all pending errors have
// been processed.
func (converter *DefaultASTConverter) errListenerFunc() func() {
	return func() {
		for {
			nextErr, more := <-converter.errChan
			if !more {
				break
			} else {
				if nextErr != nil && converter.firstErr == nil {
					converter.firstErr = nextErr
				}
			}
		}
		converter.errDone <- true
	}
}

func (converter *DefaultASTConverter) tripleListenerFunc() func() {
	return func() {
		for {
			nextTriple, more := <-converter.triplesChan
			if !more {
				break
			} else {
				err := converter.builder.HandleTriple(nextTriple)
				if err != nil {
					converter.setErr(err)
				}
			}
		}
		converter.triplesDone <- true
	}
}

func (converter *DefaultASTConverter) prefixListenerFunc() func() {
	return func() {
		for {
			nextEntry, more := <-converter.prefixChan
			if !more {
				break
			} else {
				oldValue, has := converter.namespaces[nextEntry.key]
				if has {
					converter.setErr(fmt.Errorf("Found multiple entry for namespace \"%s\": Old: \"%s\", new: \"%s\"",
						nextEntry.key, oldValue, nextEntry.value))
				} else {
					converter.namespaces[nextEntry.key] = nextEntry.value
				}
			}
		}
		converter.prefixDone <- true
	}
}

func (converter *DefaultASTConverter) getBlankNode() string {
	// first write to the channel, this will block if already there is
	// another get action
	converter.blockBlankNode <- true
	// get result
	res := converter.builder.GetBlankNode()
	// release channel
	<-converter.blockBlankNode
	return res
}

// This method simply writes the error to the errChan for the listener method.
func (converter *DefaultASTConverter) setErr(err error) {
	converter.errChan <- err
}

// This method adds a new triple to the triples channel
func (converter *DefaultASTConverter) addTriple(t *RDFTriple) {
	converter.triplesChan <- t
}

func (converter *DefaultASTConverter) addPrefix(key, value string) {
	converter.prefixChan <- &prefixEntry{key: key, value: value}
}

// The conversion is split in two parts:
// * Parse all the @prefix and the @base command
// * Parse all triples

func (converter *DefaultASTConverter) handleDirectives(directiveArcs []*ASTArc) {
	// start working functions that handle the directives
	// more details about how the workers work can be found in handleAllTriples
	workersDone := make(chan bool, converter.numStatementWorkers)
	for i := 0; i < converter.numStatementWorkers; i++ {
		workersDone <- true
	}

	foundBase := ""

	// start prefix listener
	go converter.prefixListenerFunc()()

	// start loop that iterates over all directives, waits for a free worker
	for _, directiveArc := range directiveArcs {
		if converter.firstErr != nil {
			break
		}
		// first of all: if it is a base directive, we don't start a go routine
		// but simply handle it here, there should be only one @base directive,
		// if we find two we report an error
		// this however is best done here before using another channel to
		// synchronize this...

		// again the arc we're interested in is the single child of the node
		// the arc leads to
		// TODO something seems wrong with the level...
		a := converter.ast.GetArcByID(directiveArc.Dest, 0)
		if a.ArcType.IsNonterm(NonterminalBase) {
			// TODO(Fabian) actually more than one @base is allowed:
			// 	Each @base or BASE directive sets a new In-Scope Base URI, relative to the previous one.
			// (https://www.w3.org/TR/turtle/#grammar-production-PN_PREFIX)
			// This is a BIG problem: We must parse all @base before and for each triples
			// element define which @base it is connected to
			// atm we ignore this and return an error

			// get the value it was saved with
			contentNode := converter.ast.Nodes[converter.ast.Nodes[a.Dest].Arcs[0].Dest]
			newBase := converter.ast.Tokens[contentNode.TokenRef].Seq
			// the value is saved...
			// we found a base directive, check if it was set before...
			if foundBase != "" {
				converter.setErr(fmt.Errorf("Found multiple @base directives, old value was \"%s\", new value is \"%s\"",
					foundBase, newBase))
				break
			} else {
				// set base
				foundBase = newBase
				converter.base = foundBase
			}
		} else {
			// free worker
			<-workersDone
			// start go routine
			go func(n *ASTNode) {
				converter.convertPrefix(n)
				workersDone <- true
			}(converter.ast.Nodes[a.Dest])
		}
	}
	// wait until all workers are done
	for i := 0; i < converter.numStatementWorkers; i++ {
		<-workersDone
	}
	// close prefix listener
	close(converter.prefixChan)
	// wait until all prefixes have been added
	<-converter.prefixDone
}

func (converter *DefaultASTConverter) handleAllTriples(tripleArcs []*ASTArc) {
	// start the working functions that handle the statements
	workersDone := make(chan bool, converter.numStatementWorkers)
	for i := 0; i < converter.numStatementWorkers; i++ {
		workersDone <- true
	}
	// start a loop that iterates over all statements, waits for a free worker
	// starts the handle method and reports back that a worker is now free
	for _, a := range tripleArcs {
		// before trying to parse the next statement we check for errors
		// it may be the case that some functions are still working, we don't
		// stop them but let them finish their work
		if converter.firstErr != nil {
			break
		}
		// wait for free worker
		<-workersDone
		// start go routine to handle the statement and report back
		go func(n *ASTNode) {
			state := newConverterState()
			state.base = converter.base
			// TODO here we should allow multiple base statements and
			// use the correct, using the tokenID to know which base is accurate
			converter.handleTriples(n, state)
			workersDone <- true
		}(converter.ast.Nodes[a.Dest])
	}

	// wait until all workers are done
	for i := 0; i < converter.numStatementWorkers; i++ {
		<-workersDone
	}
}

func (converter *DefaultASTConverter) Convert(ast *TurtleAST, defaultBase string, builder OWLBuilder) error {
	converter.ast = ast
	converter.builder = builder
	converter.base = defaultBase
	converter.namespaces = make(map[string]string)
	converter.bnodeLabels = make(map[string]string)
	converter.firstErr = nil
	converter.errChan = make(chan error, converter.numStatementWorkers)
	converter.errDone = make(chan bool)
	converter.triplesChan = make(chan *RDFTriple, converter.numStatementWorkers)
	converter.triplesDone = make(chan bool)
	converter.prefixChan = make(chan *prefixEntry, converter.numStatementWorkers)
	converter.prefixDone = make(chan bool)
	converter.blockBlankNode = make(chan bool, 1)

	// here the magic happens

	// start error listener method
	go converter.errListenerFunc()()
	// start triple listener
	go converter.tripleListenerFunc()()

	// first we split the tree into two parts: directives and triples
	// directives are handled first, then we parse the triples
	var directives, triples []*ASTArc

	statements := ast.Nodes[0].Arcs
	for _, stmtArc := range statements {
		// the arc we're interested in is stored (on the only element)
		// in the succ list of the child node of the statement node
		a := ast.GetArcByID(stmtArc.Dest, 0)
		if a.ArcType.IsNonterm(Directive) {
			directives = append(directives, a)
		} else {
			triples = append(triples, a)
		}
	}

	// start phase one
	converter.handleDirectives(directives)

	// check if there already was an error, if so we don't have to apply phase 2
	if converter.firstErr == nil {
		// apply phase 2
		converter.handleAllTriples(triples)
	}

	// first wait for the triples listener to finish, this may add further errors...
	close(converter.triplesChan)
	<-converter.triplesDone

	// after we're done wait for the error listener to finish and check for errors
	// first close the listener function
	close(converter.errChan)
	// wait til the channel is finished
	<-converter.errDone
	// now also close the blank node channel, all uses should be done
	close(converter.blockBlankNode)

	firstErr := converter.firstErr
	// before reporting the answer first reset the variables, return at the end
	converter.ast = nil
	converter.builder = nil
	converter.base = ""
	converter.namespaces = nil
	converter.bnodeLabels = nil
	converter.firstErr = nil
	converter.errChan = nil
	converter.errDone = nil
	converter.triplesChan = nil
	converter.triplesDone = nil
	converter.prefixChan = nil
	converter.prefixDone = nil
	converter.blockBlankNode = nil

	if firstErr != nil {
		return firstErr
	}
	return nil
}

func (converter *DefaultASTConverter) handleTriples(node *ASTNode, currentState *converterState) {
	// get first arc and determine type
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatalf("Got invalid arc in triples: %v\n", firstArc.ArcType)
	case firstArc.ArcType.IsNonterm(Subject):
		secondArc := node.Arcs[1]
		// second arc must be of type predicateObjectList
		if !secondArc.ArcType.IsNonterm(PredicateObjectList) {
			log.Fatalf("Got invalid arc in triples: %v\n", secondArc.ArcType)
		}
		// first handle the subject, this will set the state to the new subject
		currentState = converter.convertSubject(converter.ast.Nodes[firstArc.Dest], currentState)
		// handle the predicate object list
		// but first check if there already was an error, in this case do nothing
		if converter.firstErr != nil {
			return
		}
		converter.convertPredicateObjectList(converter.ast.Nodes[secondArc.Dest],
			currentState, converter.handlePOLEntryTriples)
	case firstArc.ArcType.IsNonterm(BlankNodePropertyList):
		// TODO ignored at the moment
		log.Fatal("BlankNodePropertyList not implemented")
	}
}

func (converter *DefaultASTConverter) convertSubject(node *ASTNode, currentState *converterState) *converterState {
	// first determine the type of the subject, i.e. iri | BlankNode | collection
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid subject arc")
	case firstArc.ArcType.IsNonterm(IRI):
		newState := currentState.copy()
		newState.curSubject = converter.getIRI(converter.ast.Nodes[firstArc.Dest], newState)
		return newState
	case firstArc.ArcType.IsNonterm(BlankNode):
		log.Fatal("BlankNode in subject is not supported yet")
	case firstArc.ArcType.IsNonterm(Collection):
		log.Fatal("collection in subject is not supported yet")
	}
	return nil
}

func (converter *DefaultASTConverter) getIRI(node *ASTNode, currentState *converterState) string {
	// first determine the type of the IRI, i.e. IRIREF | PrefixedName
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid IRI node")
		return ""
	case firstArc.ArcType.IsToken(IRIRef):
		contentNode := converter.ast.Nodes[firstArc.Dest]
		iri := converter.ast.Tokens[contentNode.TokenRef].Seq
		if strings.HasPrefix(iri, "#") {
			return currentState.base + iri
		} else {
			return iri
		}
	case firstArc.ArcType.IsNonterm(PrefixedName):
		prefixedNameNode := converter.ast.Nodes[firstArc.Dest]
		return converter.getPrefixedName(prefixedNameNode, currentState)
	}
}

func (converter *DefaultASTConverter) getPrefixedName(node *ASTNode, currentState *converterState) string {
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid PrefixedName node")
		return ""
	case firstArc.ArcType.IsToken(PNameLN):
		return converter.getPnameLN(converter.ast.Nodes[firstArc.Dest], currentState)
	case firstArc.ArcType.IsToken(PNameNS):
		return converter.getPNameNS(converter.ast.Nodes[firstArc.Dest], currentState)
	}
}

func (converter *DefaultASTConverter) getPNameNS(node *ASTNode, currentState *converterState) string {
	// TODO is this correct?
	val := converter.ast.Tokens[node.TokenRef].Seq
	iri, ok := converter.namespaces[val]
	if ok {
		return iri
	} else {
		converter.setErr(fmt.Errorf("No namespace for %s", val))
		return ""
	}
}

func (converter *DefaultASTConverter) getPnameLN(node *ASTNode, currentState *converterState) string {
	// TODO reservered chars?
	val := converter.ast.Tokens[node.TokenRef].Seq
	colon := strings.Index(val, ":")
	if colon < 0 {
		converter.setErr(fmt.Errorf("Invalid PNameLN (must contain \":\"): %s", val))
		return ""
	}
	prefix, localName := val[:colon], val[colon+len(":"):]
	iri, ok := converter.namespaces[prefix]
	if !ok {
		converter.setErr(fmt.Errorf("No namespace for %s", prefix))
		return ""
	}
	return iri + localName
}

// TODO move to another location?
type polEntryFunction func(verbNode, objectListNode *ASTNode, currentState *converterState)

func (converter *DefaultASTConverter) handlePOLEntryTriples(verbNode, objectListNode *ASTNode, currentState *converterState) {
	newState := currentState.copy()
	newState.curPredicate = converter.getVerb(verbNode, newState)
	// parse object list
	converter.convertObjectList(objectListNode, newState)
}

func (converter *DefaultASTConverter) convertPredicateObjectList(node *ASTNode, currentState *converterState, handleFunc polEntryFunction) {
	// we have a list of verb objectList (at least one)
	// iterate over each pair
	// we do a check that we really have a list of pairs
	if len(node.Arcs)%2 != 0 || len(node.Arcs) < 2 {
		log.Fatal("PredicateObjectList was created in an illegal way.")
	}
	var wg sync.WaitGroup
	wg.Add(len(node.Arcs) / 2)
	for i := 0; i < len(node.Arcs); i += 2 {
		go func(pos int) {
			verbArc := node.Arcs[pos]
			objectListArc := node.Arcs[pos+1]
			// do some type test s.t. only legal arcs were added
			if !verbArc.ArcType.IsNonterm(Verb) || !objectListArc.ArcType.IsNonterm(ObjectList) {
				log.Fatal("PredicateObjectList must consist of pairs \"Verb ObjectList\"")
			}
			// everything cool, call the handle function
			handleFunc(converter.ast.Nodes[verbArc.Dest], converter.ast.Nodes[objectListArc.Dest], currentState)
			// report done
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func (converter *DefaultASTConverter) convertObjectList(node *ASTNode, currentState *converterState) {
	// a list of objects (at least one)
	var wg sync.WaitGroup
	wg.Add(len(node.Arcs))
	for _, objectArc := range node.Arcs {
		go func(arc *ASTArc) {
			switch {
			default:
				log.Fatal("Arc of non object type in objectList")
			case arc.ArcType.IsNonterm(Object):
				converter.convertObject(converter.ast.Nodes[arc.Dest], currentState)
			}
			wg.Done()
		}(objectArc)
	}
	wg.Wait()
}

func (converter *DefaultASTConverter) getLiteral(node *ASTNode, currentState *converterState) interface{} {
	// TODO add other types, not only strings
	firstArc := node.Arcs[0]
	switch {
	default:
		fmt.Println(firstArc)
		log.Fatal("Only RDFLiteral is supported as literal")
		return ""
	case firstArc.ArcType.IsNonterm(RDFLiteral):
		return converter.getRDFLiteral(converter.ast.Nodes[firstArc.Dest], currentState)
	}
}

func (converter *DefaultASTConverter) getRDFLiteral(node *ASTNode, currentState *converterState) interface{} {
	// TODO no Langtag support here yet
	switch len(node.Arcs) {
	default:
		log.Fatal("Invalid number of arcs in RDFLiteral")
		return ""
	case 1:
		firstArc := node.Arcs[0]
		if !firstArc.ArcType.IsNonterm(String) {
			log.Fatal("Only String is supported right now as RDFLiteral")
			return ""
		}
		return converter.getString(converter.ast.Nodes[firstArc.Dest], currentState)
	case 3:
		log.Fatal("Langtags are not supported yet")
		return ""
	}
}

func (converter *DefaultASTConverter) getString(node *ASTNode, currentState *converterState) string {
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Only STRING_LITERAL_QUOTE is supported yet")
		return ""
	case firstArc.ArcType.IsToken(StringLiteralQuote):
		return converter.getSLQ(converter.ast.Nodes[firstArc.Dest], currentState)
	}
}

func (converter *DefaultASTConverter) getSLQ(node *ASTNode, currentState *converterState) string {
	return converter.ast.Tokens[node.TokenRef].Seq
}

func (converter *DefaultASTConverter) convertObject(node *ASTNode, currentState *converterState) {
	// a single object:
	// iri | BlankNode | collection | blankNodePropertyList | literal
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid arc in object (must be iri | BlankNode | collection | blankNodePropertyList | literal)")
	case firstArc.ArcType.IsNonterm(IRI):
		iri := converter.getIRI(converter.ast.Nodes[firstArc.Dest], currentState)
		converter.addTriple(NewRDFTriple(currentState.curSubject,
			currentState.curPredicate,
			iri))
	case firstArc.ArcType.IsNonterm(BlankNode):
		log.Fatal("BlankNode as object is not supported yet")
	case firstArc.ArcType.IsNonterm(Collection):
		matchNode := converter.convertCollection(converter.ast.Nodes[firstArc.Dest], currentState)
		// add triple with matchNode as object
		converter.addTriple(NewRDFTriple(currentState.curSubject, currentState.curPredicate, matchNode))
	case firstArc.ArcType.IsNonterm(BlankNodePropertyList):
		converter.convertBNPLFromObject(converter.ast.Nodes[firstArc.Dest],
			currentState)
	case firstArc.ArcType.IsNonterm(Literal):
		// TODO how do we store types such as int and double?
		literal := converter.getLiteral(converter.ast.Nodes[firstArc.Dest], currentState).(string)
		converter.addTriple(NewRDFTriple(currentState.curSubject,
			currentState.curPredicate,
			literal))
	}
}

// TODO is this correct? especially for empty lists...
func (converter *DefaultASTConverter) convertCollection(node *ASTNode, currentState *converterState) string {
	// collection has the form
	// object*
	if len(node.Arcs) == 0 {
		// no objects, so we simply return rdf:nil
		return RDFNil
	} else {
		// we have to create a blank node for each object, the first blank node
		// is the result
		firstBlank := converter.getBlankNode()
		newState := currentState.copy()
		newState.curSubject = firstBlank
		newState.curPredicate = RDFFirst
		// now we have to iterate over each object after the first
		for i, objectArc := range node.Arcs {
			switch {
			default:
				log.Fatal("Illegal collection production")
			case objectArc.ArcType.IsNonterm(Object):
				if i > 0 {
					// create a new blank node
					nextBlankNode := converter.getBlankNode()
					// add a triple object_n-1 rdf:rest object_n
					converter.addTriple(NewRDFTriple(newState.curSubject, RDFRest, nextBlankNode))
					// set the current subject to the new node
					newState.curSubject = nextBlankNode
				}
				// recursively call convert object
				converter.convertObject(converter.ast.Nodes[objectArc.Dest], newState.copy())
			}
		}
		// add triple curSubject rdf:rest rdf:nil
		converter.addTriple(NewRDFTriple(newState.curSubject, RDFRest, RDFNil))
		return firstBlank
	}
}

func (converter *DefaultASTConverter) handleBNPLEntryObject(verbNode, objectListNode *ASTNode, currentState *converterState) {
	verb := converter.getVerb(verbNode, currentState)
	newState := currentState.copy()
	newState.curPredicate = verb
	converter.convertObjectList(objectListNode, newState)
}

func (converter *DefaultASTConverter) convertBNPLFromObject(node *ASTNode, currentState *converterState) {
	if len(node.Arcs) != 1 {
		log.Fatal("BlankNodePropertyList must consist of exactly one predicateObjectList")
		return
	}
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid BNPL production")
	case firstArc.ArcType.IsNonterm(PredicateObjectList):
		blankNode := converter.getBlankNode()
		// TODO is this correct?
		converter.addTriple(NewRDFTriple(currentState.curSubject,
			currentState.curPredicate,
			blankNode))
		newState := currentState.copy()
		newState.curSubject = blankNode
		converter.convertPredicateObjectList(converter.ast.Nodes[firstArc.Dest],
			newState,
			converter.handleBNPLEntryObject)
	}
}

func (converter *DefaultASTConverter) getVerb(node *ASTNode, currentState *converterState) string {
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Invalid Verb production")
		return ""
	case firstArc.ArcType.IsNonterm(Predicate):
		return converter.getPredicate(converter.ast.Nodes[firstArc.Dest], currentState)
	case firstArc.ArcType.IsToken(Averb):
		return converter.getAVerb()
	}
}

func (converter *DefaultASTConverter) getAVerb() string {
	return "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
}

func (converter *DefaultASTConverter) getPredicate(node *ASTNode, currentState *converterState) string {
	firstArc := node.Arcs[0]
	switch {
	default:
		log.Fatal("Predicate must be an IRI")
		return ""
	case firstArc.ArcType.IsNonterm(IRI):
		return converter.getIRI(converter.ast.Nodes[firstArc.Dest], currentState)
	}
}

func (converter *DefaultASTConverter) convertPrefix(node *ASTNode) {
	nameID := converter.ast.Nodes[converter.ast.GetArc(node, 0).Dest].TokenRef
	irirefID := converter.ast.Nodes[converter.ast.GetArc(node, 1).Dest].TokenRef
	name := strings.TrimRight(converter.ast.Tokens[nameID].Seq, ":")
	iriRef := converter.ast.Tokens[irirefID].Seq
	converter.addPrefix(name, iriRef)
}

type ASTParser struct {
	t          TurtleTokenizer
	astBuilder ASTBuilder
	converter  ASTConverter
}

func NewASTParser(t TurtleTokenizer, astBuilder ASTBuilder, converter ASTConverter) *ASTParser {
	return &ASTParser{t: t, astBuilder: astBuilder, converter: converter}
}

func DefaultTurtleParser(numStatementWorkers int) *ASTParser {
	t := NewRegexTokenizer()
	astBuilder := NewDefaultASTBuilder()
	astConverter := NewDefaultASTConverter(numStatementWorkers)
	return &ASTParser{t: t, astBuilder: astBuilder, converter: astConverter}
}

func (parser *ASTParser) Parse(r io.Reader, defaultBase string, builder OWLBuilder) error {
	if ast, err := parser.astBuilder.BuildAST(parser.t, r); err != nil {
		return err
	} else {
		return parser.converter.Convert(ast, defaultBase, builder)
	}
}
