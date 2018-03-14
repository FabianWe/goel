// The MIT License (MIT)

// Copyright (c) 2017 Fabian Wenzelmann

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
	"log"
	"os"
	"testing"
)

func getUnivInst(builder OWLBuilder) {
	parser := DefaultTurtleParser(1)
	f, fileErr := os.Open("tests/univ-bench.ttl")
	if fileErr != nil {
		log.Fatal(fileErr)
	}
	parseErr := parser.Parse(f, "DEFAULT", builder)
	if parseErr != nil {
		log.Fatal(parseErr)
	}
}

func instTest(handler TripleQueryHandler) {
	classCount := 0
	classHandleFunc := func(t *RDFTriple) error {
		if !IsBlankNode(t.Subject) {
			classCount++
		}
		return nil
	}
	// bit ugly, but ok
	t := RDFType
	handler.AnswerQuery(nil, &t, RDFClass, classHandleFunc)
}

func TestDefaultHandler(t *testing.T) {
	builder := NewDefaultOWLBuilder()
	getUnivInst(builder)
	instTest(builder)
}
