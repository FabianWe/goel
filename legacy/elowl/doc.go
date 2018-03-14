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

/*
Package elowl builds a bridge between OWL ontology file and the internal
representation of EL++. It contains parser(s) for OWL files and builder(s)
for generating EL++ formulae given a set of RDF / OWL triples.

Parsing Input

Currently there is a parser for Turtle files (see https://www.w3.org/TR/turtle/).
However we currently don't cover the whole grammar. For example we don't support
language tags. For example the following is not supported:

    <#spiderman>
      rel:enemyOf <#green-goblin> ;
      a foaf:Person ;
      foaf:name "Spiderman", "Человек-паук"@ru .

Also there is no support for escape sequences such as \n in strings. Only
string data is supported at the moment, but the other types could be easily
added.

There are several approaches to parse these files so there are interfaces for
parsing instances (the parser interface requires a io.Reader). The default
implementation uses a hand-written tokenizer, translates the sequence of tokens
to an abstract syntax tree (AST) and from this point creates an abstract syntax
tree. Tokenization, the transforming to an AST and the transformation from an
AST to a set of triples are each defined in there own interfaces so it is easy
to plug new approaches inside the current model.

However there are some things I'm not very happy with at the moment (though
the perfomance seems ok):

* The tokenizer reads the whole file into memory before parsing the tokens.
Turtle files are very clear because each line ends a command. So it should
be ok to read the input line by line. However there are multiline texts
(''' and """) which may span more than one line. It should not be too hard
to add this speciality though. The tokenizer simply stores a list of regex
elements and the first one that matches will be the next token.
I think it a good idea to create an matcher interface that reads from the
given io.Reader. Must of them will be simple regex expressions but for
multiline strings we could use a combination of regexes and other methods.
However there should be a method to get the next line from the input in the
parser itself. This method should take care to either return the rest of
an not completely parsed line or read the next line from the input.

* No concurrency. The tokenizer and AST builder don't make use of go routines
right now which is not very nice. The converter AST --> triples however
processes several statements in a concurrent way.
*/
package elowl
