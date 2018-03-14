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
	"bytes"
	"fmt"
	"io"
	"log"
)

type ASTTypeInfo struct {
	token   TurtleToken
	nonterm TurtleNonterminal
}

func NewTypeInfoFromNameFromToken(t TurtleToken) *ASTTypeInfo {
	return &ASTTypeInfo{token: t, nonterm: -1}
}

func NewTypeInfoFromNonterm(n TurtleNonterminal) *ASTTypeInfo {
	return &ASTTypeInfo{nonterm: n, token: -1}
}

func (info *ASTTypeInfo) GetToken() (TurtleToken, bool) {
	if info.token < 0 {
		return -1, false
	}
	return info.token, true
}

func (info *ASTTypeInfo) GetNonterm() (TurtleNonterminal, bool) {
	if info.nonterm < 0 {
		return -1, false
	}
	return info.nonterm, true
}

func (info *ASTTypeInfo) IsToken(t TurtleToken) bool {
	token, _ := info.GetToken()
	return token == t
}

func (info *ASTTypeInfo) IsNonterm(n TurtleNonterminal) bool {
	nonterm, _ := info.GetNonterm()
	return nonterm == n
}

func (info *ASTTypeInfo) String() string {
	if non, isNon := info.GetNonterm(); isNon {
		return fmt.Sprintf("NonTerminal = %v", non)
	}
	if token, isToken := info.GetToken(); isToken {
		return fmt.Sprintf("Token = %v", token)
	}
	return "Invalid ASSTypeInfo"
}

type ASTArc struct {
	ArcType *ASTTypeInfo
	Dest    int
}

func NewASTArc(typeInfo *ASTTypeInfo, dest int) *ASTArc {
	return &ASTArc{ArcType: typeInfo, Dest: dest}
}

type ASTNode struct {
	TokenRef int
	Arcs     []*ASTArc
}

func NewASTNode(tokenRef int) *ASTNode {
	return &ASTNode{TokenRef: tokenRef, Arcs: nil}
}

func (node *ASTNode) AddArc(arc *ASTArc) {
	node.Arcs = append(node.Arcs, arc)
}

type TurtleAST struct {
	Nodes  []*ASTNode
	Tokens []*TokenMatch
}

func (ast *TurtleAST) GetArc(node *ASTNode, ids ...int) *ASTArc {
	var arc *ASTArc
	for _, id := range ids {
		// get arc, update node
		arc = node.Arcs[id]
		// set node to the destination of this node
		node = ast.Nodes[arc.Dest]
	}
	return arc
}

func (ast *TurtleAST) GetArcByID(nodeID int, ids ...int) *ASTArc {
	return ast.GetArc(ast.Nodes[nodeID], ids...)
}

func NewTurtleAST(tokens []*TokenMatch) *TurtleAST {
	return &TurtleAST{Nodes: nil, Tokens: tokens}
}

func (ast *TurtleAST) AddNode(node *ASTNode) int {
	n := len(ast.Nodes)
	ast.Nodes = append(ast.Nodes, node)
	return n
}

func (ast *TurtleAST) Backtrack(nodeID int) {
	for i := nodeID; i < len(ast.Nodes); i++ {
		ast.Nodes[i] = nil
	}
	ast.Nodes = ast.Nodes[:nodeID]
}

func (ast *TurtleAST) String() string {
	w := bytes.NewBufferString("")
	for i, node := range ast.Nodes {
		if node == nil {
			fmt.Fprintln(w, "removed node")
			continue
		}
		fmt.Fprintf(w, "%d (%v) -->\n", i, node.TokenRef)
		for _, arc := range node.Arcs {
			arcType := "nil"
			arcDest := "nil"
			if arc.ArcType != nil {
				arcType = fmt.Sprintf("%v", arc.ArcType)
			}
			if arc.Dest >= 0 {
				arcDest = fmt.Sprintf("%d", arc.Dest)
			}
			fmt.Fprintf(w, "\t%s: %s\n", arcType, arcDest)
		}
	}
	return w.String()
}

type ASTBuilder interface {
	BuildAST(t TurtleTokenizer, r io.Reader) (*TurtleAST, error)
}

type DefaultASTBuilder struct {
	t      TurtleTokenizer
	tokens []*TokenMatch
	ast    *TurtleAST
}

func NewDefaultASTBuilder() *DefaultASTBuilder {
	return &DefaultASTBuilder{t: nil, tokens: nil, ast: nil}
}

func (builder *DefaultASTBuilder) BuildAST(t TurtleTokenizer, r io.Reader) (*TurtleAST, error) {
	builder.t = t
	builder.tokens = nil
	builder.ast = NewTurtleAST(nil)
	if err := builder.t.Init(r); err != nil {
		return nil, fmt.Errorf("Error while tokenization: %v", err)
	}
	if err := builder.readAllTokens(); err != nil {
		return nil, fmt.Errorf("Error while reading tokens: %v", err)
	}
	_, _, err := builder.handleTurtleDoc()
	if err != nil {
		return nil, err
	}
	builder.ast.Tokens = builder.tokens
	return builder.ast, nil
}

func (builder *DefaultASTBuilder) expectErr(expected string, tm *TokenMatch) error {
	var errOut string
	if len(tm.Seq) > 100 {
		errOut = fmt.Sprintf("%.100q...", tm.Seq)
	} else {
		errOut = fmt.Sprintf("%q", tm.Seq)
	}
	return fmt.Errorf("Expected %s but got token type %v while parsing text %s", expected, tm.Token, errOut)
}

func (builder *DefaultASTBuilder) readAllTokens() error {
	for {
		tm, err := builder.t.NextToken()
		if err != nil {
			return err
		}
		switch tm.Token {
		default:
			builder.tokens = append(builder.tokens, tm)
		case Comment, WS:
			// ignore
		case EOF:
			return nil
		case ErrorToken:
			var errOut string
			if len(tm.Seq) > 100 {
				errOut = fmt.Sprintf("%.100q...", tm.Seq)
			} else {
				errOut = tm.Seq
			}
			return fmt.Errorf("Error while tokenization. Error sequence: %s", errOut)
		}
	}
}

func (builder *DefaultASTBuilder) nextToken(next int) *TokenMatch {
	if next >= len(builder.tokens) {
		return nil
	}
	return builder.tokens[next]
}

func (builder *DefaultASTBuilder) handleToken(next, parent int, t TurtleToken) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	if token.Token != t {
		return next, -1, builder.expectErr(fmt.Sprintf("%v", t), token)
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)
	return next + 1, nodeID, nil
}

type astHandlerFunc func(next, parent int) (int, int, error)

func (builder *DefaultASTBuilder) firstMatch(next, parent int, funcs ...astHandlerFunc) (int, int, int) {
	for i, f := range funcs {
		if new, nodeID, err := f(next, parent); err == nil {
			return new, nodeID, i
		}
	}
	return next, -1, -1
}

func (builder *DefaultASTBuilder) matchAll(next, parent int, funcs ...astHandlerFunc) (int, []int) {
	res := make([]int, len(funcs))
	new := next
	for i, f := range funcs {
		// TODO is this call correct? should be...
		if nextNew, nodeID, err := f(new, parent); err != nil {
			return next, nil
		} else {
			res[i] = nodeID
			new = nextNew
		}
	}
	return new, res
}

func (builder *DefaultASTBuilder) handleIRIRef(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, IRIRef)
}

func (builder *DefaultASTBuilder) handlePrefixDirective(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, PrefixDirective)
}

func (builder *DefaultASTBuilder) handleBaseDirective(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, BaseDirective)
}

func (builder *DefaultASTBuilder) handlePNAmeNS(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, PNameNS)
}

func (builder *DefaultASTBuilder) handleBlankNodeLabel(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, BlankNodeLabel)
}

func (builder *DefaultASTBuilder) handleStringLiteralQuote(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, StringLiteralQuote)
}

func (builder *DefaultASTBuilder) handleAnnon(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, Annon)
}

func (builder *DefaultASTBuilder) handlePNameLN(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, PNameLN)
}

func (builder *DefaultASTBuilder) handlePoint(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, Point)
}

func (builder *DefaultASTBuilder) handleOpenBrace(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, OpenBrace)
}

func (builder *DefaultASTBuilder) handleCloseBrace(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, CloseBrace)
}

func (builder *DefaultASTBuilder) handleOpenBracket(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, OpenBracket)
}

func (builder *DefaultASTBuilder) handleCloseBracket(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, CloseBracket)
}

func (builder *DefaultASTBuilder) handleOpenCurlyBrace(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, OpenCurlyBrace)
}

func (builder *DefaultASTBuilder) handleCloseCurlyBrace(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, CloseCurlyBrace)
}

func (builder *DefaultASTBuilder) handleComma(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, Comma)
}

func (builder *DefaultASTBuilder) handleSemicolon(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, Semicolon)
}

func (builder *DefaultASTBuilder) handleAverb(next, parent int) (int, int, error) {
	return builder.handleToken(next, parent, Averb)
}

func (builder *DefaultASTBuilder) handleTurtleDoc() (int, int, error) {
	// in this case we simply create a new node containing the root and add
	// as many statements as we may find
	root := NewASTNode(0)
	next := 0
	builder.ast.AddNode(root)
	var child int
	var err error
	for {
		// try to parse statement
		next, child, err = builder.handleStatement(next, 0)
		if err == ErrNoToken {
			// in this case we're done!
			// everything is fine
			return next, 0, nil
		}
		// check if err != nil, if yes another error has occured
		if err != nil {
			return 0, -1, err
		}
		// now everything is fine, so we add a child node
		a := NewASTArc(NewTypeInfoFromNonterm(Statement), child)
		root.AddArc(a)
	}
}

func (builder *DefaultASTBuilder) handleNonterminalPrefixID(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)
	// we have to parse @prefix PNAME_NS IRIREF .
	newNext, res := builder.matchAll(next, nodeID,
		builder.handlePrefixDirective,
		builder.handlePNAmeNS,
		builder.handleIRIRef,
		builder.handlePoint)
	if res == nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("@prefix PNAME_NS IRIREF .", token)
	}
	// everything has matched, so add child nodes...
	if len(res) != 4 {
		log.Fatal("Weird return value in handleNonterminalPrefixID")
	}
	// we actually ignore the @prefix, it isn't required later on
	// the same holds for the ., no need to save it
	builder.ast.Nodes[res[0]] = nil
	builder.ast.Nodes[res[3]] = nil
	// add arcs
	tInfoName := NewTypeInfoFromNameFromToken(PNameNS)
	tInfoIRI := NewTypeInfoFromNameFromToken(IRIRef)
	arc1 := NewASTArc(tInfoName, res[1])
	arc2 := NewASTArc(tInfoIRI, res[2])
	node.AddArc(arc1)
	node.AddArc(arc2)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleNonterminalBase(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)
	// we have to parse @base IRIREF .
	newNext, res := builder.matchAll(next, nodeID,
		builder.handleBaseDirective,
		builder.handleIRIRef,
		builder.handlePoint)
	if res == nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("@base IRIREF .", token)
	}
	// everything has matched, so add child nodes...
	if len(res) != 3 {
		log.Fatal("Weird return value in handleNonterminalPrefixID")
	}
	// again ignore @base and the .
	builder.ast.Nodes[res[0]] = nil
	builder.ast.Nodes[res[2]] = nil
	// add arcs
	tInfoIRI := NewTypeInfoFromNameFromToken(IRIRef)
	arc1 := NewASTArc(tInfoIRI, res[1])
	node.AddArc(arc1)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleDirective(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try both possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleNonterminalBase,
		builder.handleNonterminalPrefixID)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("prefixID or base", token)
	}
	// else we have a match
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNonterm(NonterminalBase)
	case 1:
		tInfo = NewTypeInfoFromNonterm(NonterminalPrefixID)
	default:
		log.Fatal("Weird return value in handlePrefixedName")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleSubject(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try both possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleIRI,
		builder.handleBlankNode,
		builder.handleCollection)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("iri | BlankNode | collection", token)
	}
	// else we have a match
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNonterm(IRI)
	case 1:
		tInfo = NewTypeInfoFromNonterm(BlankNode)
	case 2:
		tInfo = NewTypeInfoFromNonterm(Collection)
	default:
		log.Fatal("Weird return value in handleSubject")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleIRI(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try both possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleIRIRef, builder.handlePrefixedName)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("IRIREF or PrefixedName", token)
	}
	// else we have a match, if 0 add node to child with IRIREF
	// otherwise add to child with PrefixedName
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNameFromToken(IRIRef)
	case 1:
		tInfo = NewTypeInfoFromNonterm(PrefixedName)
	default:
		log.Fatal("Weird return value in handleIRI")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleBlankNode(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleBlankNodeLabel, builder.handleAnnon)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("BLANK_NODE_LABEL or ANNON", token)
	}
	// else we have a match, if 0 add node to child with blank node label
	// otherwise add to child with annon
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNameFromToken(BlankNodeLabel)
	case 1:
		tInfo = NewTypeInfoFromNameFromToken(Annon)
	default:
		log.Fatal("Weird return value in handleBlankNode")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) Bla(r io.Reader) {
	if err := builder.t.Init(r); err != nil {
		log.Fatal(err)
	}
	if err := builder.readAllTokens(); err != nil {
		log.Fatal(err)
	}
	builder.handleVerb(0, 0)
}

func (builder *DefaultASTBuilder) handleCollection(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	// first we must match (
	nextNew, child, err := builder.handleOpenBrace(next, nodeID)
	// we don't add the child node... ( is ignored
	if err != nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("(", token)
	}
	// we have a match... now read objects as long as possible
	for {
		nextNew, child, err = builder.handleObject(nextNew, nodeID)
		// if there was an error during parsing no problem, maybe we just
		// finished the objects?
		// break the loop and expect )
		if err != nil {
			break
		}
		// we finally parsed a new object, so add a child node
		tInfo := NewTypeInfoFromNonterm(Object)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
	}
	// we're outside the loop because there was a break
	// so now we have to check if on the last position (aka the one that lead
	// to an error) there is a closing )
	nextNew, child, err = builder.handleCloseBrace(nextNew, nodeID)
	// if there is an error we don't have a valid collection
	if err != nil {
		// backtrack
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr(")", token)
	}
	// everything is ok here
	return nextNew, nodeID, nil
}

func (builder *DefaultASTBuilder) handlePrefixedName(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try both possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handlePNameLN, builder.handlePNAmeNS)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("PNAME_LN or PNAME_NS", token)
	}
	// else we have a match, if 0 add node to child with PNAME_LN
	// otherwise add to child with PNAME_NS
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNameFromToken(PNameLN)
	case 1:
		tInfo = NewTypeInfoFromNameFromToken(PNameNS)
	default:
		log.Fatal("Weird return value in handlePrefixedName")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

// subject predicateObjectList | blankNodePropertyList predicateObjectList?
// save the folowing child nodes: either subject and predicateObjectList
// or blankNodePropertyList and predicateObjectList or just blankNodePropertyList
func (builder *DefaultASTBuilder) handleTriples(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	newNext, res := builder.matchAll(next, nodeID,
		builder.handleSubject,
		builder.handlePredicateObjectList)
	if res != nil {
		// it's ok, we add the nodes
		if len(res) != 2 {
			log.Fatal("Weird return value in handleTriples")
		}
		a1 := NewASTArc(NewTypeInfoFromNonterm(Subject), res[0])
		a2 := NewASTArc(NewTypeInfoFromNonterm(PredicateObjectList), res[1])
		node.AddArc(a1)
		node.AddArc(a2)
	} else {
		// now we have to match blankNodePropertyList predicateObjectList?
		var child int
		var err error
		newNext, child, err = builder.handleBlankNodePropertyList(newNext, nodeID)
		// on error: return error
		if err != nil {
			builder.ast.Backtrack(nodeID)
			return next, -1, builder.expectErr("blankNodePropertyList", token)
		}
		// parsing did work... so now add
		a1 := NewASTArc(NewTypeInfoFromNonterm(BlankNodePropertyList), child)
		node.AddArc(a1)
		// so we now may parse a predicateObjectList
		newNext, child, err = builder.handlePredicateObjectList(newNext, nodeID)
		// if this did not work, well no problem...
		// only if there was no error we do something
		if err == nil {
			a2 := NewASTArc(NewTypeInfoFromNonterm(PredicateObjectList), nodeID)
			node.AddArc(a2)
		}
	}
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleObject(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try all possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleIRI,
		builder.handleBlankNode,
		builder.handleCollection,
		builder.handleBlankNodePropertyList,
		builder.handleLiteral)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("iri | BlankNode | collection | blankNodePropertyList | literal", token)
	}
	// else we have a match
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNonterm(IRI)
	case 1:
		tInfo = NewTypeInfoFromNonterm(BlankNode)
	case 2:
		tInfo = NewTypeInfoFromNonterm(Collection)
	case 3:
		tInfo = NewTypeInfoFromNonterm(BlankNodePropertyList)
	case 4:
		tInfo = NewTypeInfoFromNonterm(Literal)
	default:
		log.Fatal("Weird return value in handleObject")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleBlankNodePropertyList(next, parent int) (int, int, error) {
	// we must match [ predicateObjectList ]
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	newNext, res := builder.matchAll(next, nodeID,
		builder.handleOpenBracket,
		builder.handlePredicateObjectList,
		builder.handleCloseBracket)
	if res == nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("[ predicateObjectList ]", token)
	}
	// everything has matched, so add child nodes...
	if len(res) != 3 {
		log.Fatal("Weird return value in handleNonterminalPrefixID")
	}
	// we ignore the brackets
	builder.ast.Nodes[res[0]] = nil
	builder.ast.Nodes[res[2]] = nil
	// we must match [ predicateObjectList ]
	arc := NewASTArc(NewTypeInfoFromNonterm(PredicateObjectList), res[1])
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handleLiteral(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleRDFLiteral)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("literal", token)
	}
	if match == 0 {
		tInfo := NewTypeInfoFromNonterm(RDFLiteral)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
		return newNext, nodeID, nil
	} else {
		log.Fatal("Weird return value in handleLiteral")
		// will never be reached because of log.Fatal, but well go complains otherwise
		return -1, -1, nil
	}
}

func (builder *DefaultASTBuilder) handleString(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleStringLiteralQuote)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("String", token)
	}
	if match == 0 {
		tInfo := NewTypeInfoFromNameFromToken(StringLiteralQuote)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
		return newNext, nodeID, nil
	} else {
		log.Fatal("Weird return value in handleString")
		// will never be reached because of log.Fatal, but well go complains otherwise
		return -1, -1, nil
	}
}

func (builder *DefaultASTBuilder) handleRDFLiteral(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleString)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("RDFLiteral", token)
	}
	if match == 0 {
		tInfo := NewTypeInfoFromNonterm(String)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
		return newNext, nodeID, nil
	} else {
		log.Fatal("Weird return value in handleRDFLiteral")
		// will never be reached because of log.Fatal, but well go complains otherwise
		return -1, -1, nil
	}
}

func (builder *DefaultASTBuilder) handleObjectList(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	// first we must match an object
	nextNew, child, err := builder.handleObject(next, nodeID)
	// on error return
	if err != nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("object", token)
	}
	// add object
	firstArc := NewASTArc(NewTypeInfoFromNonterm(Object), child)
	node.AddArc(firstArc)
	// we have a match... now read , andobjects as long as possible
	for {
		// first we must match a ,
		nextNew, child, err = builder.handleComma(nextNew, nodeID)
		// on error break the loop!
		// everything is fine in this case
		if err != nil {
			break
		}
		// we had a , so now there must be an object
		// on error backtrack and return with an error
		nextNew, child, err = builder.handleObject(nextNew, nodeID)
		if err != nil {
			builder.ast.Backtrack(nodeID)
		}
		// we finally parsed a new object, so add a child node
		tInfo := NewTypeInfoFromNonterm(Object)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
	}
	// once we're here everything is ok
	return nextNew, nodeID, nil
}

// The rule has the form
// verb objectList (';' (verb objectList)?)*
// So we add the following child nodes:
// - first pair of verb objectList
// - for each match in *: verb objectList or nil, -1 and nil, -1
func (builder *DefaultASTBuilder) handlePredicateObjectList(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	// first we must match a verb
	nextNew, child, err := builder.handleVerb(next, nodeID)
	// on error return
	if err != nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("verb", token)
	}
	// add it
	tInfo := NewTypeInfoFromNonterm(Verb)
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)

	// now we must match an objectList
	nextNew, child, err = builder.handleObjectList(nextNew, nodeID)
	// on error return
	if err != nil {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("objectList", token)
	}
	// add the object list
	objectListArc := NewASTArc(NewTypeInfoFromNonterm(ObjectList), child)
	node.AddArc(objectListArc)
	// everything good so far, now parse the list
	for {
		// first we must match a ;
		nextNew, child, err = builder.handleSemicolon(nextNew, nodeID)
		// if it is not a ; break the loop, we're done in this case
		if err != nil {
			break
		}
		// again, don't store ;
		// now we *may* match a pair verb objectList
		nextNew, child, err = builder.handleVerb(nextNew, nodeID)
		// if we had no success we simply add nil, nil
		if err != nil {
			arc1 := NewASTArc(nil, -1)
			arc2 := NewASTArc(nil, -1)
			node.AddArc(arc1)
			node.AddArc(arc2)
		} else {
			// before we do anything else: create new arc2
			a1 := NewASTArc(NewTypeInfoFromNonterm(Verb), child)
			node.AddArc(a1)
			// if we had success we *must* parse an objectList
			nextNew, child, err = builder.handleObjectList(nextNew, nodeID)
			// on error return
			if err != nil {
				builder.ast.Backtrack(nodeID)
				return next, -1, builder.expectErr("objectList", token)
			} else {
				// add second arc as well
				a2 := NewASTArc(NewTypeInfoFromNonterm(ObjectList), child)
				node.AddArc(a2)
			}
		}
	}
	// here everything is ok
	return nextNew, nodeID, nil
}

func (builder *DefaultASTBuilder) handleVerb(next, parent int) (int, int, error) {
	// crate new node
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	// create new node and try both possible values

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handlePredicate,
		builder.handleAverb)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("predicate or a", token)
	}
	// else we have a match, if 0 add node to child with preidcate
	// otherwise add to child with Averb
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNonterm(Predicate)
	case 1:
		tInfo = NewTypeInfoFromNameFromToken(Averb)
	default:
		log.Fatal("Weird return value in handleVerb")
	}
	// add arc from this node to the child
	arc := NewASTArc(tInfo, child)
	node.AddArc(arc)
	return newNext, nodeID, nil
}

func (builder *DefaultASTBuilder) handlePredicate(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)

	nodeID := builder.ast.AddNode(node)
	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleIRI)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("iri", token)
	}
	if match == 0 {
		tInfo := NewTypeInfoFromNonterm(IRI)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
		return newNext, nodeID, nil
	} else {
		log.Fatal("Weird return value in handlePredicate")
		// will never be reached because of log.Fatal, but well go complains otherwise
		return -1, -1, nil
	}
}

func (builder *DefaultASTBuilder) handleStatement(next, parent int) (int, int, error) {
	token := builder.nextToken(next)
	if token == nil {
		return next, -1, ErrNoToken
	}
	node := NewASTNode(next)
	nodeID := builder.ast.AddNode(node)

	newNext, child, match := builder.firstMatch(next, nodeID,
		builder.handleDirective,
		builder.handleTriples)
	// check if match is < 0: In this case no match was found, Backtrack
	// and return error
	if match < 0 {
		builder.ast.Backtrack(nodeID)
		return next, -1, builder.expectErr("directive or triples .", token)
	}
	// else we have a match
	var tInfo *ASTTypeInfo
	switch match {
	case 0:
		tInfo = NewTypeInfoFromNonterm(Directive)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
	case 1:
		tInfo = NewTypeInfoFromNonterm(Triples)
		arc := NewASTArc(tInfo, child)
		node.AddArc(arc)
		// now we expect a point
		var err error
		newNext, child, err = builder.handlePoint(newNext, nodeID)
		if err != nil {
			// backtrack and return error
			builder.ast.Backtrack(nodeID)
			return next, -1, builder.expectErr(".", token)
		}
		// we found a point, but we don't add it
	default:
		log.Fatal("Weird return value in handlePrefixedName")
	}
	return newNext, nodeID, nil
}
