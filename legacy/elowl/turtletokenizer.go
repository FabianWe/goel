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
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
)

// Defines the basic char sets. This is however not complete with respect
// to the turtle grammar definition. Note that these constants are meant to
// be used in regular expressions, so for example the - string is escaped.
const (
	pnBase  = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
	pnU     = pnBase + `_`
	pnChars = pnU + `\-0123456789`
)

// Defines a type for turtle grammar objects. The list is nearly as defined
// in the turtle documentation. Some small changes have been made, for example
// @base and @prefix are defined as tokens.
type TurtleToken int

const (
	ErrorToken TurtleToken = iota
	EOF
	WS
	Comment
	IRIRef
	PNameNS
	BlankNodeLabel
	StringLiteralQuote
	Annon
	PNameLN
	Point
	OpenBrace
	CloseBrace
	OpenBracket
	CloseBracket
	OpenCurlyBrace
	CloseCurlyBrace
	Comma
	Semicolon
	Averb
	PrefixDirective
	BaseDirective
)

// A human readable version of the token name.
func (t TurtleToken) String() string {
	switch t {
	case ErrorToken:
		return "ERROR"
	case EOF:
		return "EOF"
	case WS:
		return "WS"
	case Comment:
		return "Comment"
	case IRIRef:
		return "IRIREF"
	case PNameNS:
		return "PNAME_NS"
	case BlankNodeLabel:
		return "BLANK_NODE_LABEL"
	case StringLiteralQuote:
		return "STRING_LITERAL_QUOTE"
	case Annon:
		return "ANON"
	case PNameLN:
		return "PNAME_LN"
	case Point:
		return "."
	case OpenBrace:
		return "("
	case CloseBrace:
		return ")"
	case OpenBracket:
		return "["
	case CloseBracket:
		return "]"
	case OpenCurlyBrace:
		return "{"
	case CloseCurlyBrace:
		return "}"
	case Comma:
		return ","
	case Semicolon:
		return ";"
	case Averb:
		return "a"
	case BaseDirective:
		return "base directive"
	case PrefixDirective:
		return "prefix directive"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", t)
	}
}

func (t TurtleToken) RegexCapture() string {
	switch t {
	case ErrorToken:
		return "error"
	case EOF:
		return "eof"
	case WS:
		return "ws"
	case Comment:
		return "comment"
	case IRIRef:
		return "iriref"
	case PNameNS:
		return "pnamens"
	case BlankNodeLabel:
		return "blanknodelabel"
	case StringLiteralQuote:
		return "slq"
	case Annon:
		return "anon"
	case PNameLN:
		return "pnameln"
	case Point:
		return "point"
	case OpenBrace:
		return "lbrace"
	case CloseBrace:
		return "rbrace"
	case OpenBracket:
		return "lbracket"
	case CloseBracket:
		return "rbracket"
	case OpenCurlyBrace:
		return "lcurly"
	case CloseCurlyBrace:
		return "rcurly"
	case Comma:
		return "comma"
	case Semicolon:
		return "semicolon"
	case Averb:
		return "a"
	case BaseDirective:
		return "base"
	case PrefixDirective:
		return "prefix"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", t)
	}
}

func TokenFromRegexCapture(str string) TurtleToken {
	switch str {
	case "lbracket":
		return OpenBracket
	case "rbracket":
		return CloseBracket
	case "rbrace":
		return CloseBrace
	case "blanknodelabel":
		return BlankNodeLabel
	case "prefix":
		return PrefixDirective
	case "rcurly":
		return CloseCurlyBrace
	case "lbrace":
		return OpenBrace
	case "comment":
		return Comment
	case "anon":
		return Annon
	case "slq":
		return StringLiteralQuote
	case "base":
		return BaseDirective
	case "ws":
		return WS
	case "iriref":
		return IRIRef
	case "a":
		return Averb
	case "point":
		return Point
	case "lcurly":
		return OpenCurlyBrace
	case "comma":
		return Comma
	case "semicolon":
		return Semicolon
	case "pnameln":
		return PNameLN
	case "pnamens":
		return PNameNS
	default:
		panic(fmt.Sprintf("Unkown turtle regex capture: \"%s\"", str))
	}
}

// A match defines the toke type of the match and the string sequence with
// which it was matched.
type TokenMatch struct {
	Token TurtleToken
	Seq   string
}

func (match *TokenMatch) CleanUp() {
	s := match.Seq
	switch match.Token {
	case ErrorToken, EOF, WS, Comment:
		// do nothing
	case IRIRef:
		s = strings.TrimLeft(strings.TrimRight(strings.TrimSpace(s), ">"), "<")
	case BlankNodeLabel:
		// TODO check
		s = strings.TrimSpace(s)
	case StringLiteralQuote:
		s = strings.Trim(strings.TrimSpace(s), "\"")
	default:
		// TODO Check if for the rest trim space is enough
		s = strings.TrimSpace(match.Seq)
	}
	match.Seq = s
}

/*
An interface for functions that return a sequence of tokens.
Each time before you use a tokenizer you *must* call its Init method. The
tokenizer should also be able to handle subsequent calls with different
readers, i.e. you can reuse it for tokenizing another file.
However before you use NextToken you must always call Init.
After that the tokenizer returns the next Token by calling NextToken.
Note one important thing about the NextToken method: An error != nil should
only be returned if there was an error while reading the input file. If a
syntax error occurred you return error = nil but the token ErrorToken.
*/
type TurtleTokenizer interface {
	Init(r io.Reader) error
	NextToken() (*TokenMatch, error)
}

// A token ca be matched with a regex. This class stores the regex for the token
// and the type of the token matchted by this regex.
type tokenRegexMatcher struct {
	token TurtleToken
	m     *regexMatcher
}

// Simply stores the regex, this type is here because we could plug in an
// interface that specifies a match method that doesn't require only
// on a regex. This type however does.
type regexMatcher struct {
	regex *regexp.Regexp
}

// The match function returns a []string. It is supposed to return something
// like the regex match functions of golang. So it returns nil if no match
// was found and a slice all the matches otherwise (in regexp these are the
// groups). The most important thing is that the first element should contain
// the whole string that was matched.
func (rm *regexMatcher) match(s string) []string {
	return rm.regex.FindStringSubmatch(s)
}

// Tokenizes a reader by trying a list of regexes, the first that matches
// is the next token. Implements the tokenizer interface.
type RegexTokenizer struct {
	regexes []*tokenRegexMatcher
	s       string
}

func (t *RegexTokenizer) Init(r io.Reader) error {
	allContent, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	t.s = string(allContent)
	return nil
}

// Add a new regex and register it with a certain token type.
func (t *RegexTokenizer) addRegex(token TurtleToken, regex *regexp.Regexp) {
	rm := regexMatcher{regex: regex}
	info := tokenRegexMatcher{token: token, m: &rm}
	t.regexes = append(t.regexes, &info)
}

// Add a prefix regex. This method adds `^` in front of the prefix and
// `(\s|$)` after the prefix. This means "Match only at beginning of string
// and afterwards expect a whitespace or the end of the file".
func (t *RegexTokenizer) addPrefixRegex(token TurtleToken, prefix string) {
	re := regexp.MustCompile(`^` + prefix + `(\s|$)`)
	t.addRegex(token, re)
}

func NewRegexTokenizer() *RegexTokenizer {
	res := RegexTokenizer{regexes: nil, s: ""}

	pnprefix := `[%s](([%s\.])*[%s])?`
	pnprefix = fmt.Sprintf(pnprefix, pnBase, pnChars, pnChars)

	pnamensStr := `(%s)?\:`
	pnamensStr = fmt.Sprintf(pnamensStr, pnprefix)

	pnlocalescStr := `\\[_~\.\-!\$&'\(\)\*\+,;=/\?#@%]`

	// simplified!
	plxStr := pnlocalescStr

	pnlocalStr := `([%s:0-9]|(%s))(([%s]|(%s))*([%s:]|(%s)))?`
	pnlocalStr = fmt.Sprintf(pnlocalStr, pnU, plxStr, pnChars, plxStr, pnChars, plxStr)

	echar := `\\["'tbnrf\\]`

	wsR := regexp.MustCompile(`^\s+`)
	res.addRegex(WS, wsR)

	// comment simply checks for # and moves on until the end of the line
	commentR := regexp.MustCompile(`^#[^\n]*(\n|$)`)
	res.addRegex(Comment, commentR)

	irirefR := regexp.MustCompile(`^<([^<>"{}|^\\]+)>(\s|$)`)
	res.addRegex(IRIRef, irirefR)

	blankRaw := `^_:([%s0-9])(([%s\.])*[%s])?(\s|$)`
	blankRaw = fmt.Sprintf(blankRaw, pnU, pnChars, pnChars)
	blankNodeLabelR := regexp.MustCompile(blankRaw)
	res.addRegex(BlankNodeLabel, blankNodeLabelR)

	pnamelnStr := `^(%s)(%s)(\s|$)`
	pnamelnStr = fmt.Sprintf(pnamelnStr, pnamensStr, pnlocalStr)
	res.addRegex(PNameLN, regexp.MustCompile(pnamelnStr))

	pnamensR := regexp.MustCompile(fmt.Sprintf(`^%s(\s|$)`, pnamensStr))
	res.addRegex(PNameNS, pnamensR)

	// NOTE @ is not allowed by the syntax, but our benchmark use it...well
	// so I simply added it
	// PS the same for . and /
	// seems to do no harm...
	// oh seems - must be allowed inside a string as well
	// oh, seems that there are no numbers allowed in strings?
	// but why?
	// Tere is are a number of other things that were not allowed in strings...
	// changed this. But too many to enumerate them all...
	// maybe we should just match everything expcept "...
	// TODO But strings are even simpler, "" strings are never allowed to contain
	// an "
	slqRaw := fmt.Sprintf(`^"(%s|[%s/\.@\-0-9,;:]|\s)*"(\s|$)`, echar, pnU)
	stringliteralquoteR := regexp.MustCompile(slqRaw)
	res.addRegex(StringLiteralQuote, stringliteralquoteR)

	annonR := regexp.MustCompile(`^\[\s*\](\s|$)`)
	res.addRegex(Annon, annonR)

	// Looks correct, but I'm not sure...
	averbR := regexp.MustCompile(`^a(\s|$)`)
	res.addRegex(Averb, averbR)

	baseDirectiveR := regexp.MustCompile(`^@base(\s|$)`)
	res.addRegex(BaseDirective, baseDirectiveR)

	prefixDirectiveR := regexp.MustCompile(`^@prefix(\s|$)`)
	res.addRegex(PrefixDirective, prefixDirectiveR)

	// add handlers for . [ ] { } , ; ( )
	res.addPrefixRegex(Point, `\.`)
	res.addPrefixRegex(OpenBracket, `\[`)
	res.addPrefixRegex(CloseBracket, `\]`)
	res.addPrefixRegex(OpenCurlyBrace, `\{`)
	res.addPrefixRegex(CloseCurlyBrace, `\}`)
	res.addPrefixRegex(Comma, ",")
	res.addPrefixRegex(Semicolon, ";")
	res.addPrefixRegex(OpenBrace, `\(`)
	res.addPrefixRegex(CloseBrace, `\)`)
	return &res
}

func (t *RegexTokenizer) NextToken() (*TokenMatch, error) {
	if t.s == "" {
		return &TokenMatch{EOF, ""}, nil
	}
	token := ErrorToken
	seq := t.s
	for _, info := range t.regexes {
		match := info.m.match(seq)
		if match != nil {
			token = info.token
			seq = match[0]
			t.s = t.s[len(seq):]
			break
		}
	}
	res := &TokenMatch{Token: token, Seq: seq}
	res.CleanUp()
	return res, nil
}

type OneRegexTokenizer struct {
	r *regexp.Regexp
	// captures the index of a group and maps it to the token it belongs to
	tokenMap map[int]TurtleToken
	s        string
}

func (t *OneRegexTokenizer) registerRegex(token TurtleToken, regexStr string) string {
	return fmt.Sprintf(`(?P<%s>%s)`, token.RegexCapture(), regexStr)
}

func (t *OneRegexTokenizer) prefixRegex(token TurtleToken, prefix string) string {
	return t.registerRegex(token, `^`+prefix+`(\s|$)`)
}

func (t *OneRegexTokenizer) Init(r io.Reader) error {
	allContent, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	t.s = string(allContent)
	return nil
}

func NewOneRegexTokenizer() *OneRegexTokenizer {
	tokenMap := make(map[int]TurtleToken)
	res := OneRegexTokenizer{r: nil, s: "", tokenMap: tokenMap}
	rs := make([]string, 0, 20)

	pnprefix := `[%s](([%s\.])*[%s])?`
	pnprefix = fmt.Sprintf(pnprefix, pnBase, pnChars, pnChars)

	pnamensStr := `(%s)?\:`
	pnamensStr = fmt.Sprintf(pnamensStr, pnprefix)

	pnlocalescStr := `\\[_~\.\-!\$&'\(\)\*\+,;=/\?#@%]`

	// simplified!
	plxStr := pnlocalescStr

	pnlocalStr := `([%s:0-9]|(%s))(([%s]|(%s))*([%s:]|(%s)))?`
	pnlocalStr = fmt.Sprintf(pnlocalStr, pnU, plxStr, pnChars, plxStr, pnChars, plxStr)

	echar := `\\["'tbnrf\\]`

	wsR := `^\s+`
	rs = append(rs, res.registerRegex(WS, wsR))

	// comment simply checks for # and moves on until the end of the line
	commentR := `^#[^\n]*(\n|$)`
	rs = append(rs, res.registerRegex(Comment, commentR))

	irirefR := `^<([^<>"{}|^\\]+)>(\s|$)`
	rs = append(rs, res.registerRegex(IRIRef, irirefR))

	blankRaw := `^_:([%s0-9])(([%s\.])*[%s])?(\s|$)`
	blankRaw = fmt.Sprintf(blankRaw, pnU, pnChars, pnChars)
	rs = append(rs, res.registerRegex(BlankNodeLabel, blankRaw))

	pnamelnStr := `^(%s)(%s)(\s|$)`
	pnamelnStr = fmt.Sprintf(pnamelnStr, pnamensStr, pnlocalStr)
	rs = append(rs, res.registerRegex(PNameLN, pnamelnStr))

	pnamensR := fmt.Sprintf(`^%s(\s|$)`, pnamensStr)
	rs = append(rs, res.registerRegex(PNameNS, pnamensR))

	// NOTE @ is not allowed by the syntax, but our benchmark use it...well
	// so I simply added it
	// PS the same for . and /
	// seems to do no harm...
	// oh seems - must be allowed inside a string as well
	// oh, seems that there are no numbers allowed in strings?
	// but why?
	// Tere is are a number of other things that were not allowed in strings...
	// changed this. But too many to enumerate them all...
	// maybe we should just match everything expcept "...
	// TODO But strings are even simpler, "" strings are never allowed to contain
	// an "
	slqRaw := fmt.Sprintf(`^"(%s|[%s/\.@\-0-9,;:]|\s)*"(\s|$)`, echar, pnU)
	rs = append(rs, res.registerRegex(StringLiteralQuote, slqRaw))

	annonR := `^\[\s*\](\s|$)`
	rs = append(rs, res.registerRegex(Annon, annonR))

	// Looks correct, but I'm not sure...
	averbR := `^a(\s|$)`
	rs = append(rs, res.registerRegex(Averb, averbR))

	baseDirectiveR := `^@base(\s|$)`
	rs = append(rs, res.registerRegex(BaseDirective, baseDirectiveR))

	prefixDirectiveR := `^@prefix(\s|$)`
	rs = append(rs, res.registerRegex(PrefixDirective, prefixDirectiveR))

	// add handlers for . [ ] { } , ; ( )
	rs = append(rs, res.prefixRegex(Point, `\.`))
	rs = append(rs, res.prefixRegex(OpenBracket, `\[`))
	rs = append(rs, res.prefixRegex(CloseBracket, `\]`))
	rs = append(rs, res.prefixRegex(OpenCurlyBrace, `\{`))
	rs = append(rs, res.prefixRegex(CloseCurlyBrace, `\}`))
	rs = append(rs, res.prefixRegex(Comma, ","))
	rs = append(rs, res.prefixRegex(Semicolon, ";"))
	rs = append(rs, res.prefixRegex(OpenBrace, `\(`))
	rs = append(rs, res.prefixRegex(CloseBrace, `\)`))

	regexStr := fmt.Sprintf("(%s)", strings.Join(rs, "|"))
	res.r = regexp.MustCompile(regexStr)
	for i, name := range res.r.SubexpNames() {
		if i != 0 && name != "" {
			res.tokenMap[i] = TokenFromRegexCapture(name)
		}
	}
	return &res
}

func (t *OneRegexTokenizer) NextToken() (*TokenMatch, error) {
	if t.s == "" {
		return &TokenMatch{EOF, ""}, nil
	}
	// match next token
	match := t.r.FindStringSubmatch(t.s)
	if match == nil {
		return &TokenMatch{Token: ErrorToken, Seq: t.s}, nil
	}

	token := ErrorToken
	seq := t.s

	for i, tokenType := range t.tokenMap {
		if matchStr := match[i]; matchStr != "" {
			seq = matchStr
			t.s = t.s[len(seq):]
			token = tokenType
			break
		}
	}
	res := &TokenMatch{Token: token, Seq: seq}
	res.CleanUp()
	return res, nil
}

func (t *OneRegexTokenizer) Match(str string) {
	match := t.r.FindStringSubmatch(str)
	res := make(map[string]string)
	for i, name := range t.r.SubexpNames() {
		if i != 0 {
			res[name] = match[i]
		}
	}
}

type SynchTokenizer struct {
	*RegexTokenizer
}

func (t *SynchTokenizer) NextToken() (*TokenMatch, error) {
	if t.s == "" {
		return &TokenMatch{EOF, ""}, nil
	}
	seq := t.s
	// for each regex start its own go routine
	// save result in slice
	matches := make([]*TokenMatch, len(t.regexes))
	var wg sync.WaitGroup
	wg.Add(len(t.regexes))
	for i := 0; i < len(t.regexes); i++ {
		go func(regexID int) {
			info := t.regexes[regexID]
			match := info.m.match(seq)
			if match != nil {
				token := info.token
				matchedStr := match[0]
				tm := &TokenMatch{Token: token, Seq: matchedStr}
				tm.CleanUp()
				matches[regexID] = tm
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	// check for first match, keep the order!
	for _, nextMatch := range matches {
		if nextMatch == nil {
			continue
		}
		t.s = t.s[len(nextMatch.Seq):]
		tm := &TokenMatch{Token: nextMatch.Token, Seq: nextMatch.Seq}
		tm.CleanUp()
		return tm, nil
	}
	return &TokenMatch{Token: ErrorToken, Seq: t.s}, nil
}
