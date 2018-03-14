// The MIT License (MIT)

// Copyright (c) 2016 Fabian Wenzelmann

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
	"strings"
	"testing"
)

func initTokenizer(s string) *RegexTokenizer {
	l := NewRegexTokenizer()
	r := strings.NewReader(s)
	l.Init(r)
	return l
}

func TestTokenizerEOF(t *testing.T) {
	l := initTokenizer("")
	m, _ := l.NextToken()
	if m.Token != EOF {
		t.Errorf("Expected EOF, got %v", m.Token)
	}
}

func TestWS(t *testing.T) {
	l := initTokenizer("  ")
	m, _ := l.NextToken()
	if m.Token != WS || m.Seq != "  " {
		t.Errorf("Expected whitespace and string \"  \", got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerDirective(t *testing.T) {
	l := initTokenizer("@prefix")
	m, _ := l.NextToken()
	if m.Token != PrefixDirective || m.Seq != "@prefix" {
		t.Errorf("Expected type DIRECTIVE and result @prefix, but got %v and %s", m.Token, m.Seq)
	}

}

func TestTokenizerDirectiveTwo(t *testing.T) {
	l := initTokenizer("@base")
	m, _ := l.NextToken()
	if m.Token != BaseDirective || m.Seq != "@base" {
		t.Errorf("Expected type DIRECTIVE and result @base, but got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerDirectiveThree(t *testing.T) {
	l := initTokenizer("@foo")
	m, _ := l.NextToken()
	if m.Token != ErrorToken {
		t.Errorf("Expected tokenizer to return error, got %v", m.Token)
	}
}

func TestTokenizerIRIRef(t *testing.T) {
	l := initTokenizer("<http://swat.cse.lehigh.edu/onto/univ-bench.owl#>")
	m, _ := l.NextToken()
	if m.Token != IRIRef || m.Seq != "http://swat.cse.lehigh.edu/onto/univ-bench.owl#" {
		t.Errorf("Expected IRIREF and \"http://swat.cse.lehigh.edu/onto/univ-bench.owl#\", got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerPNameNS(t *testing.T) {
	l := initTokenizer(":")
	m, _ := l.NextToken()
	if m.Token != PNameNS || m.Seq != ":" {
		t.Errorf("Expected PNAME_NS and \":\", got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerPNameNSTwo(t *testing.T) {
	l := initTokenizer("owl:")
	m, _ := l.NextToken()
	if m.Token != PNameNS || m.Seq != "owl:" {
		t.Errorf("Expected PNAME_NS and \"owl:\", got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerBlankNodeLabel(t *testing.T) {
	l := initTokenizer("_:genid42")
	m, _ := l.NextToken()
	if m.Token != BlankNodeLabel || m.Seq != "_:genid42" {
		t.Errorf("Expected BLANK_NODE_LABEL and \"_:genid42\", got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerStringLiteralQuote(t *testing.T) {
	l := initTokenizer(`"Foo"`)
	m, _ := l.NextToken()
	if m.Token != StringLiteralQuote || m.Seq != `Foo` {
		t.Errorf(`Expected STRING_LITERAL_QUOTE and Foo, got %v and %s`, m.Token, m.Seq)
	}
}

func TestTokenizerStringLiteralQuoteTwo(t *testing.T) {
	inp := `"Foo\"\\\nBar"`
	l := initTokenizer(inp)
	m, _ := l.NextToken()
	if m.Token != StringLiteralQuote || m.Seq != `Foo\"\\\nBar` {
		t.Errorf(`Expected STRING_LITERAL_QUOTE and %s, got %v and %s`, inp, m.Token, m.Seq)
	}
}

func TestTokenizerStringLiteralQuoteThree(t *testing.T) {
	inp := `"Foo\"`
	l := initTokenizer(inp)
	m, _ := l.NextToken()
	if m.Token != ErrorToken {
		t.Errorf("Expected ERROR, got %v and %s", m.Token, m.Seq)
	}
}

func TestTokenizerAnnon(t *testing.T) {
	inp := "[    ]"
	l := initTokenizer(inp)
	m, _ := l.NextToken()
	if m.Token != Annon || m.Seq != inp {
		t.Errorf("Expected ANON and %s, got %v and %s", inp, m.Token, m.Seq)
	}
}

func TestOpenBracket(t *testing.T) {
	l := initTokenizer("[ foo]")
	m, _ := l.NextToken()
	if m.Token != OpenBracket || m.Seq != "[" {
		t.Errorf("Expected [ and [, got %v and %s", m.Token, m.Seq)
	}
}
