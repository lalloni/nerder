// Package snippet holds the Go sample shown on every font card and a
// go/scanner-based highlighter that turns it into classed spans.
package snippet

import (
	"go/scanner"
	"go/token"
	"strings"
)

// Span is a run of source text with a highlight class: keyword, string,
// number, comment, operator, punct, builtin, func, ident or plain.
type Span struct {
	Class string `json:"class"`
	Text  string `json:"text"`
}

// Source is valid Go that touches most of the language and every
// ligature-prone operator, plus a few Nerd Font glyphs.
var Source = strings.NewReplacer(
	"{GO}", "", "{CODE}", "", "{ARROW}", "", "{TICK}", "`",
).Replace(`package main

// ligatures:  :=  <-  ->  !=  ==  <=  >=  &&  ||  ++
//            --  ...  +=  -=  &^  <<  >>  //  /*  */  =>  ::

import (
	"context"
	"fmt"
	"sync"
)

/* A tiny worker pool {GO} */
const (
	Idle State = iota
	Busy
)

type State int

type Job struct {
	ID   int      {TICK}json:"id"{TICK}
	Tags []string {TICK}json:"tags,omitempty"{TICK}
}

type Runner interface {
	Run(ctx context.Context, j Job) error
}

type Queue[T any] struct {
	mu    sync.Mutex
	items []T
}

func (q *Queue[T]) Push(items ...T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, items...)
}

func work(ctx context.Context, jobs <-chan Job, r Runner) error {
	for j := range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := r.Run(ctx, j); err != nil {
				return fmt.Errorf("job %d: %w", j.ID, err)
			}
		}
	}
	return nil
}

func kind(v any) string {
	switch x := v.(type) {
	case int:
		return "int {CODE} " + fmt.Sprint(x)
	case error:
		return "error"
	}
	return "other"
}

func main() {
	jobs := make(chan Job, 8)
	seen := map[string]bool{"a": true}
	go func() {
		defer close(jobs)
		for i := 0; i <= 3 && i >= 0 || len(seen) == 0; i++ {
			jobs <- Job{ID: i}
		}
	}()
	// pipeline -> done {ARROW} 0xFF 3.14 'x'
	err := work(context.Background(), jobs, nil)
	if err != nil {
		fmt.Println("failed:", err)
	}
}
`)

var predeclared = map[string]bool{
	"any": true, "bool": true, "byte": true, "comparable": true, "complex64": true,
	"complex128": true, "error": true, "float32": true, "float64": true, "int": true,
	"int8": true, "int16": true, "int32": true, "int64": true, "rune": true,
	"string": true, "uint": true, "uint8": true, "uint16": true, "uint32": true,
	"uint64": true, "uintptr": true, "true": true, "false": true, "iota": true,
	"nil": true, "append": true, "cap": true, "clear": true, "close": true,
	"complex": true, "copy": true, "delete": true, "imag": true, "len": true,
	"make": true, "max": true, "min": true, "new": true, "panic": true,
	"print": true, "println": true, "real": true, "recover": true,
}

// Highlight splits src into spans whose texts concatenate back to src.
// Adjacent operator characters (such as - and >) share one span so font
// ligatures can form across them.
func Highlight(src string) []Span {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, scanner.ScanComments)

	var spans []Span
	pos := 0
	add := func(class, text string) {
		if text == "" {
			return
		}
		if n := len(spans); n > 0 && class == "operator" && spans[n-1].Class == "operator" {
			spans[n-1].Text += text
			return
		}
		spans = append(spans, Span{class, text})
	}
	for {
		p, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		off := file.Offset(p)
		if tok == token.SEMICOLON && lit != ";" {
			continue // automatically inserted; the newline is part of the next gap
		}
		text := lit
		if text == "" {
			text = tok.String()
		}
		add("plain", src[pos:off])
		end := off + len(text)
		add(classify(tok, text, src[end:]), src[off:end])
		pos = end
	}
	add("plain", src[pos:])
	return spans
}

func classify(tok token.Token, text, after string) string {
	switch {
	case tok == token.COMMENT:
		return "comment"
	case tok == token.STRING || tok == token.CHAR:
		return "string"
	case tok == token.INT || tok == token.FLOAT || tok == token.IMAG:
		return "number"
	case tok == token.IDENT:
		switch {
		case predeclared[text]:
			return "builtin"
		case strings.HasPrefix(after, "("):
			return "func"
		}
		return "ident"
	case tok.IsKeyword():
		return "keyword"
	case tok == token.LPAREN, tok == token.RPAREN, tok == token.LBRACK, tok == token.RBRACK,
		tok == token.LBRACE, tok == token.RBRACE, tok == token.COMMA, tok == token.SEMICOLON,
		tok == token.PERIOD, tok == token.COLON:
		return "punct"
	case tok.IsOperator():
		return "operator"
	}
	return "plain"
}
