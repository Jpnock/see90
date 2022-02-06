package c90

import (
	"bufio"
	"io"
	"strings"
)

type frame struct {
	i            int
	s            string
	line, column int
}
type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch      chan frame
	ch_stop chan bool
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []frame
	stale bool

	// The 'l' and 'c' fields were added for
	// https://github.com/wagerlabs/docker/blob/65694e801a7b80930961d70c69cba9f2465459be/buildfile.nex
	// Since then, I introduced the built-in Line() and Column() functions.
	l, c int

	parseResult interface{}

	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan frame)
	yylex.ch_stop = make(chan bool, 1)
	var scan func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int)
	scan = func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int) {
		// Index of DFA and length of highest-precedence match so far.
		matchi, matchn := 0, -1
		var buf []rune
		n := 0
		checkAccept := func(i int, st int) bool {
			// Higher precedence match? DFAs are run in parallel, so matchn is at most len(buf), hence we may omit the length equality check.
			if family[i].acc[st] && (matchn < n || matchi > i) {
				matchi, matchn = i, n
				return true
			}
			return false
		}
		var state [][2]int
		for i := 0; i < len(family); i++ {
			mark := make([]bool, len(family[i].startf))
			// Every DFA starts at state 0.
			st := 0
			for {
				state = append(state, [2]int{i, st})
				mark[st] = true
				// As we're at the start of input, follow all ^ transitions and append to our list of start states.
				st = family[i].startf[st]
				if -1 == st || mark[st] {
					break
				}
				// We only check for a match after at least one transition.
				checkAccept(i, st)
			}
		}
		atEOF := false
		stopped := false
		for {
			if n == len(buf) && !atEOF {
				r, _, err := in.ReadRune()
				switch err {
				case io.EOF:
					atEOF = true
				case nil:
					buf = append(buf, r)
				default:
					panic(err)
				}
			}
			if !atEOF {
				r := buf[n]
				n++
				var nextState [][2]int
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if -1 == x[1] {
						continue
					}
					nextState = append(nextState, x)
					checkAccept(x[0], x[1])
				}
				state = nextState
			} else {
			dollar: // Handle $.
				for _, x := range state {
					mark := make([]bool, len(family[x[0]].endf))
					for {
						mark[x[1]] = true
						x[1] = family[x[0]].endf[x[1]]
						if -1 == x[1] || mark[x[1]] {
							break
						}
						if checkAccept(x[0], x[1]) {
							// Unlike before, we can break off the search. Now that we're at the end, there's no need to maintain the state of each DFA.
							break dollar
						}
					}
				}
				state = nil
			}

			if state == nil {
				lcUpdate := func(r rune) {
					if r == '\n' {
						line++
						column = 0
					} else {
						column++
					}
				}
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					lcUpdate(buf[0])
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1
					select {
					case ch <- frame{matchi, text, line, column}:
						{
						}
					case stopped = <-ch_stop:
						{
						}
					}
					if stopped {
						break
					}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, ch_stop, family[matchi].nest, line, column)
					}
					if atEOF {
						break
					}
					for _, r := range text {
						lcUpdate(r)
					}
				}
				n = 0
				for i := 0; i < len(family); i++ {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		ch <- frame{-1, "", line, column}
	}
	go scan(bufio.NewReader(in), yylex.ch, yylex.ch_stop, dfas, 0, 0)
	return yylex
}

type dfa struct {
	acc          []bool           // Accepting states.
	f            []func(rune) int // Transitions.
	startf, endf []int            // Transitions at start and end of input.
	nest         []dfa
}

var dfas = []dfa{
	// auto
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return 1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 111:
				return -1
			case 116:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 111:
				return 4
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// break
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return 1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return 3
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 4
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return 5
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 101:
				return -1
			case 107:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// case
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return 1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 2
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return 4
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// char
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return 1
			case 104:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 104:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 3
			case 99:
				return -1
			case 104:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 104:
				return -1
			case 114:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 104:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// const
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 99:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// continue
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 99:
				return 1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return 5
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// default
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 6
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return 7
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// do
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 100:
				return 1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// double
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return 1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return 4
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return 5
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return 6
			case 108:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 98:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// else
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return 1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 108:
				return 2
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 4
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 108:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// enum
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return 1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 109:
				return 4
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// extern
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return 1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 120:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 4
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 120:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// float
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 102:
				return 1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 102:
				return -1
			case 108:
				return 2
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 111:
				return 3
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 4
			case 102:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// for
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 102:
				return 1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 102:
				return -1
			case 111:
				return 2
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 102:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// goto
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 103:
				return 1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 111:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 111:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// if
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 102:
				return -1
			case 105:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 102:
				return 2
			case 105:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 102:
				return -1
			case 105:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// int
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 105:
				return 1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return 2
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// long
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 108:
				return 1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return 4
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 103:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// register
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return 1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 2
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return 3
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 4
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return 5
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 7
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return 8
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// return
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 2
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return 5
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return 6
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 110:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// short
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 104:
				return 2
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 104:
				return -1
			case 111:
				return 3
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return 4
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 104:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// signed
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 2
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return 3
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 4
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return 5
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return 6
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// sizeof
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 115:
				return 1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return 2
			case 111:
				return -1
			case 115:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 122:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 4
			case 102:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 111:
				return 5
			case 115:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return 6
			case 105:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 102:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 115:
				return -1
			case 122:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// static
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 3
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return 5
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return 6
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 99:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// struct
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return 5
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// switch
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 115:
				return 1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 119:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return -1
			case 105:
				return 3
			case 115:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return 5
			case 104:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return 6
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 99:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// typedef
	{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return 1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 112:
				return 3
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return 4
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return 5
			case 101:
				return -1
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return 6
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return 7
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 112:
				return -1
			case 116:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// union
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return 2
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return 3
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return 5
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 105:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// unsigned
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 2
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return 3
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return 4
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return 5
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return 6
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return 7
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return 8
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 105:
				return -1
			case 110:
				return -1
			case 115:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// void
	{[]bool{false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 105:
				return -1
			case 111:
				return 2
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 105:
				return 3
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return 4
			case 105:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 100:
				return -1
			case 105:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// volatile
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return 2
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 4
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return 6
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return 7
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return 8
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// while
	{[]bool{false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 104:
				return 2
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return 3
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return 4
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return 5
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1}, nil},

	// [a-zA-Z_]([a-zA-Z_]|[0-9])*
	{[]bool{false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 95:
				return 1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return 1
			case 97 <= r && r <= 122:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 95:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// 0[xX][a-fA-F0-9]+((u|U)|(u|U)?(l|L|ll|LL)|(l|L|ll|LL)(u|U))?
	{[]bool{false, false, false, true, true, true, true, true, true, true, true, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 48:
				return 1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return 2
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 3
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 3
			case 76:
				return 4
			case 85:
				return 5
			case 88:
				return -1
			case 108:
				return 6
			case 117:
				return 7
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 15
			case 85:
				return 12
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return 14
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 8
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return 9
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 12
			case 88:
				return -1
			case 108:
				return 13
			case 117:
				return 14
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 8
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return 9
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 11
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return 10
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 12
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return 14
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 12
			case 88:
				return -1
			case 108:
				return -1
			case 117:
				return 14
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// 0[0-7]*((u|U)|(u|U)?(l|L|ll|LL)|(l|L|ll|LL)(u|U))?
	{[]bool{false, true, true, true, true, true, true, true, true, true, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 48:
				return 1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 2
			case 76:
				return 3
			case 85:
				return 4
			case 108:
				return 5
			case 117:
				return 6
			}
			switch {
			case 48 <= r && r <= 55:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return 2
			case 76:
				return 3
			case 85:
				return 4
			case 108:
				return 5
			case 117:
				return 6
			}
			switch {
			case 48 <= r && r <= 55:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 14
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 7
			case 85:
				return -1
			case 108:
				return 8
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return 12
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 7
			case 85:
				return -1
			case 108:
				return 8
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return 10
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return 9
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 55:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [1-9][0-9]*((u|U)|(u|U)?(l|L|ll|LL)|(l|L|ll|LL)(u|U))?
	{[]bool{false, true, true, true, true, true, true, true, true, true, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 2
			case 85:
				return 3
			case 108:
				return 4
			case 117:
				return 5
			}
			switch {
			case 48 <= r && r <= 48:
				return 6
			case 49 <= r && r <= 57:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 14
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 7
			case 85:
				return -1
			case 108:
				return 8
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return 12
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 7
			case 85:
				return -1
			case 108:
				return 8
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 2
			case 85:
				return 3
			case 108:
				return 4
			case 117:
				return 5
			}
			switch {
			case 48 <= r && r <= 48:
				return 6
			case 49 <= r && r <= 57:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 10
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return 9
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 117:
				return -1
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 85:
				return 11
			case 108:
				return -1
			case 117:
				return 13
			}
			switch {
			case 48 <= r && r <= 48:
				return -1
			case 49 <= r && r <= 57:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// L?'(\\.|[^\\'\n])+'
	{[]bool{false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return 1
			case 76:
				return 2
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return -1
			case 76:
				return 3
			case 92:
				return 4
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return 1
			case 76:
				return -1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return 6
			case 76:
				return 3
			case 92:
				return 4
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 10:
				return 5
			case 39:
				return 5
			case 76:
				return 5
			case 92:
				return 5
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return 6
			case 76:
				return 3
			case 92:
				return 4
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 39:
				return -1
			case 76:
				return -1
			case 92:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// [0-9]+([Ee][+-]?[0-9]+)(f|F|l|L)?
	{[]bool{false, false, false, false, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return 2
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 3
			case 45:
				return 3
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return 5
			case 76:
				return 6
			case 101:
				return -1
			case 102:
				return 7
			case 108:
				return 8
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [0-9]*\.[0-9]+([Ee][+-]?[0-9]+)?(f|F|l|L)?
	{[]bool{false, false, true, false, true, true, true, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 0
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return 3
			case 70:
				return 4
			case 76:
				return 5
			case 101:
				return 3
			case 102:
				return 6
			case 108:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 8
			case 45:
				return 8
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 9
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 9
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return 4
			case 76:
				return 5
			case 101:
				return -1
			case 102:
				return 6
			case 108:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return 9
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [0-9]+\.[0-9]*([Ee][+-]?[0-9]+)?(f|F|l|L)?
	{[]bool{false, false, true, false, true, true, true, true, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 2
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return 3
			case 70:
				return 4
			case 76:
				return 5
			case 101:
				return 3
			case 102:
				return 6
			case 108:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 9
			case 45:
				return 9
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 10
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return 3
			case 70:
				return 4
			case 76:
				return 5
			case 101:
				return 3
			case 102:
				return 6
			case 108:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 10
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 69:
				return -1
			case 70:
				return 4
			case 76:
				return 5
			case 101:
				return -1
			case 102:
				return 6
			case 108:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return 10
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// 0[xX][a-fA-F0-9]+([Pp][+-]?[0-9]+)(f|F|l|L)?
	{[]bool{false, false, false, false, false, false, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 3
			case 70:
				return 3
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 3
			case 70:
				return 3
			case 76:
				return -1
			case 80:
				return 4
			case 88:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 112:
				return 4
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 5
			case 45:
				return 5
			case 48:
				return 6
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 6
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 6
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 6
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return 6
			case 70:
				return 7
			case 76:
				return 8
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 6
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// 0[xX][a-fA-F0-9]*\.[a-fA-F0-9]+([Pp][+-]?[0-9]+)?(f|F|l|L)?
	{[]bool{false, false, false, false, false, true, true, true, false, true, true, false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 3
			case 48:
				return 4
			case 70:
				return 4
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 4
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			case 65 <= r && r <= 70:
				return 4
			case 97 <= r && r <= 102:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 5
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 5
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 3
			case 48:
				return 4
			case 70:
				return 4
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 4
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 4
			case 65 <= r && r <= 70:
				return 4
			case 97 <= r && r <= 102:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 11
			case 45:
				return 11
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return 13
			case 76:
				return 7
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 14
			case 108:
				return 10
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// 0[xX][a-fA-F0-9]+\.[a-fA-F0-9]*([Pp][+-]?[0-9]+)?(f|F|l|L)?
	{[]bool{false, false, false, false, true, true, true, true, false, true, true, false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return 2
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return 2
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 3
			case 70:
				return 3
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return 4
			case 48:
				return 3
			case 70:
				return 3
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 3
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 3
			case 65 <= r && r <= 70:
				return 3
			case 97 <= r && r <= 102:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 11
			case 45:
				return 11
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 5
			case 70:
				return 6
			case 76:
				return 7
			case 80:
				return 8
			case 88:
				return -1
			case 102:
				return 9
			case 108:
				return 10
			case 112:
				return 8
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 5
			case 65 <= r && r <= 70:
				return 5
			case 97 <= r && r <= 102:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return 12
			case 70:
				return 13
			case 76:
				return 7
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return 14
			case 108:
				return 10
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 12
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 46:
				return -1
			case 48:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 80:
				return -1
			case 88:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 112:
				return -1
			case 120:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 70:
				return -1
			case 97 <= r && r <= 102:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// L?\"(\\.|[^\\"\n])*\"
	{[]bool{false, false, false, true, false, false, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return 1
			case 76:
				return 2
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return 3
			case 76:
				return 4
			case 92:
				return 5
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return 1
			case 76:
				return -1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return -1
			case 76:
				return -1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return 3
			case 76:
				return 4
			case 92:
				return 5
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 10:
				return 6
			case 34:
				return 6
			case 76:
				return 6
			case 92:
				return 6
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 34:
				return 3
			case 76:
				return 4
			case 92:
				return 5
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// \.\.\.
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 46:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 46:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 46:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 46:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// >>=
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 3
			case 62:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// <<=
	{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return 2
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// \+=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \-=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 45:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \*=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \/=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 47:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 47:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 47:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// %=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 37:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// &=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 38:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 38:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 38:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \^=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 94:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			case 94:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 94:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \|=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 124:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 124:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// >>
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 62:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// <<
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \+\+
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \-\-
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 45:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \->
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 45:
				return 1
			case 62:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			case 62:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// &&
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 38:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 38:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 38:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// \|\|
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 124:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 124:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 124:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// <=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// >=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			case 62:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// ==
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// !=
	{[]bool{false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			case 61:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 61:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// ;
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 59:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 59:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// {
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 123:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 123:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// }
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 125:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 125:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// ,
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 44:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 44:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// :
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 58:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 58:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// =
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 61:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 61:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \(
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 40:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 40:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \)
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 41:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \[
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 91:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 91:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \]
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 93:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 93:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \.
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 46:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 46:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// &
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 38:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 38:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// !
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// ~
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 126:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 126:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \-
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 45:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 45:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \+
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \*
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \/
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 47:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// %
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 37:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// <
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 60:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 60:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// >
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 62:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 62:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \^
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 94:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 94:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \|
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 124:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 124:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \?
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 63:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 63:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// [ \t\v\n\f]
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 10:
				return 1
			case 11:
				return 1
			case 12:
				return 1
			case 32:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 9:
				return -1
			case 10:
				return -1
			case 11:
				return -1
			case 12:
				return -1
			case 32:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// .
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			return 1
		},
		func(r rune) int {
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},
}

func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}

func (yyLex *Lexer) Stop() {
	yyLex.ch_stop <- true
}

// Text returns the matched text.
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}

// Line returns the current line number.
// The first line is 0.
func (yylex *Lexer) Line() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].line
}

// Column returns the current column number.
// The first column is 0.
func (yylex *Lexer) Column() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].column
}

func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		l, c := 0, 0
		if lvl > 0 {
			l, c = yylex.stack[lvl-1].line, yylex.stack[lvl-1].column
		}
		yylex.stack = append(yylex.stack, frame{0, "", l, c})
	}
	if lvl == len(yylex.stack)-1 {
		p := &yylex.stack[lvl]
		*p = <-yylex.ch
		yylex.stale = false
	} else {
		yylex.stale = true
	}
	return yylex.stack[lvl].i
}
func (yylex *Lexer) pop() {
	yylex.stack = yylex.stack[:len(yylex.stack)-1]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}

// Lex runs the lexer. Always returns 0.
// When the -s option is given, this function is not generated;
// instead, the NN_FUN macro runs the lexer.
func (yylex *Lexer) Lex(lval *yySymType) int {
OUTER0:
	for {
		switch yylex.next(0) {
		case 0:
			{
				count(yylex)
				return AUTO
			}
		case 1:
			{
				count(yylex)
				return BREAK
			}
		case 2:
			{
				count(yylex)
				return CASE
			}
		case 3:
			{
				count(yylex)
				return CHAR
			}
		case 4:
			{
				count(yylex)
				return CONST
			}
		case 5:
			{
				count(yylex)
				return CONTINUE
			}
		case 6:
			{
				count(yylex)
				return DEFAULT
			}
		case 7:
			{
				count(yylex)
				return DO
			}
		case 8:
			{
				count(yylex)
				return DOUBLE
			}
		case 9:
			{
				count(yylex)
				return ELSE
			}
		case 10:
			{
				count(yylex)
				return ENUM
			}
		case 11:
			{
				count(yylex)
				return EXTERN
			}
		case 12:
			{
				count(yylex)
				return FLOAT
			}
		case 13:
			{
				count(yylex)
				return FOR
			}
		case 14:
			{
				count(yylex)
				return GOTO
			}
		case 15:
			{
				count(yylex)
				return IF
			}
		case 16:
			{
				count(yylex)
				return INT
			}
		case 17:
			{
				count(yylex)
				return LONG
			}
		case 18:
			{
				count(yylex)
				return REGISTER
			}
		case 19:
			{
				count(yylex)
				return RETURN
			}
		case 20:
			{
				count(yylex)
				return SHORT
			}
		case 21:
			{
				count(yylex)
				return SIGNED
			}
		case 22:
			{
				count(yylex)
				return SIZEOF
			}
		case 23:
			{
				count(yylex)
				return STATIC
			}
		case 24:
			{
				count(yylex)
				return STRUCT
			}
		case 25:
			{
				count(yylex)
				return SWITCH
			}
		case 26:
			{
				count(yylex)
				return TYPEDEF
			}
		case 27:
			{
				count(yylex)
				return UNION
			}
		case 28:
			{
				count(yylex)
				return UNSIGNED
			}
		case 29:
			{
				count(yylex)
				return VOID
			}
		case 30:
			{
				count(yylex)
				return VOLATILE
			}
		case 31:
			{
				count(yylex)
				return WHILE
			}
		case 32:
			{
				count(yylex)
				lval.str = yylex.Text()
				return checkType()
			}
		case 33:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 34:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 35:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 36:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 37:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 38:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 39:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 40:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 41:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 42:
			{
				count(yylex)
				lval.str = yylex.Text()
				return CONSTANT
			}
		case 43:
			{
				count(yylex)
				return STRING_LITERAL
			}
		case 44:
			{
				count(yylex)
				return ELLIPSIS
			}
		case 45:
			{
				count(yylex)
				return RIGHT_ASSIGN
			}
		case 46:
			{
				count(yylex)
				return LEFT_ASSIGN
			}
		case 47:
			{
				count(yylex)
				return ADD_ASSIGN
			}
		case 48:
			{
				count(yylex)
				return SUB_ASSIGN
			}
		case 49:
			{
				count(yylex)
				return MUL_ASSIGN
			}
		case 50:
			{
				count(yylex)
				return DIV_ASSIGN
			}
		case 51:
			{
				count(yylex)
				return MOD_ASSIGN
			}
		case 52:
			{
				count(yylex)
				return AND_ASSIGN
			}
		case 53:
			{
				count(yylex)
				return XOR_ASSIGN
			}
		case 54:
			{
				count(yylex)
				return OR_ASSIGN
			}
		case 55:
			{
				count(yylex)
				return RIGHT_OP
			}
		case 56:
			{
				count(yylex)
				return LEFT_OP
			}
		case 57:
			{
				count(yylex)
				return INC_OP
			}
		case 58:
			{
				count(yylex)
				return DEC_OP
			}
		case 59:
			{
				count(yylex)
				return PTR_OP
			}
		case 60:
			{
				count(yylex)
				return AND_OP
			}
		case 61:
			{
				count(yylex)
				return OR_OP
			}
		case 62:
			{
				count(yylex)
				return LE_OP
			}
		case 63:
			{
				count(yylex)
				return GE_OP
			}
		case 64:
			{
				count(yylex)
				return EQ_OP
			}
		case 65:
			{
				count(yylex)
				return NE_OP
			}
		case 66:
			{
				count(yylex)
				return int(';')
			}
		case 67:
			{
				count(yylex)
				return int(123)
			}
		case 68:
			{
				count(yylex)
				return int(125)
			}
		case 69:
			{
				count(yylex)
				return int(',')
			}
		case 70:
			{
				count(yylex)
				return int(':')
			}
		case 71:
			{
				count(yylex)
				return int('=')
			}
		case 72:
			{
				count(yylex)
				return int('(')
			}
		case 73:
			{
				count(yylex)
				return int(')')
			}
		case 74:
			{
				count(yylex)
				return int('[')
			}
		case 75:
			{
				count(yylex)
				return int(']')
			}
		case 76:
			{
				count(yylex)
				return int('.')
			}
		case 77:
			{
				count(yylex)
				return int('&')
			}
		case 78:
			{
				count(yylex)
				return int('!')
			}
		case 79:
			{
				count(yylex)
				return int('~')
			}
		case 80:
			{
				count(yylex)
				return int('-')
			}
		case 81:
			{
				count(yylex)
				return int('+')
			}
		case 82:
			{
				count(yylex)
				return int('*')
			}
		case 83:
			{
				count(yylex)
				return int('/')
			}
		case 84:
			{
				count(yylex)
				return int('%')
			}
		case 85:
			{
				count(yylex)
				return int('<')
			}
		case 86:
			{
				count(yylex)
				return int('>')
			}
		case 87:
			{
				count(yylex)
				return int('^')
			}
		case 88:
			{
				count(yylex)
				return int('|')
			}
		case 89:
			{
				count(yylex)
				return int('?')
			}
		case 90:
			{
				count(yylex)
			}
		case 91:
			{
			}
		default:
			break OUTER0
		}
		continue
	}
	yylex.pop()

	return 0
}

var (
	column = 0
)

func count(yylex *Lexer) {
	for _, c := range yylex.Text() {
		if c == '\n' {
			column = 0
		} else if c == '\t' {
			column += 8 - (column % 8)
		} else {
			column += 1
		}
	}
	return
}

func checkType() int {
	// TODO: check if a typedef already exists in the current scope. If
	// it does, return TYPE_NAME instead.
	return IDENTIFIER
}
