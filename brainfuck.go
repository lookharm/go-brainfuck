package main

import (
	"fmt"
	"io"
)

type OpCode int

const (
	INCP OpCode = iota
	DECP
	INC
	DEC
	OUT
	IN
	JMPF
	JMPB
)

var CharacterToOpCode = map[rune]OpCode{
	'>': INCP,
	'<': DECP,
	'+': INC,
	'-': DEC,
	'.': OUT,
	',': IN,
	'[': JMPF,
	']': JMPB,
}

var OpCodeToCharacter = map[OpCode]rune{
	INCP: '>',
	DECP: '<',
	INC:  '+',
	DEC:  '-',
	OUT:  '.',
	IN:   ',',
	JMPF: '[',
	JMPB: ']',
}

type JumpTable [8]ExecuteFunction

type ExecuteFunction func(pc, dp, ic *int, context *Context) *byte

// TODO: Input must be stream
var jumpTable [8]ExecuteFunction = [8]ExecuteFunction{
	INCP: func(pc, dp, ic *int, context *Context) *byte {
		*dp++
		if *dp >= len(context.data) {
			panic(fmt.Sprintf("brainfuck: invalid data pointer, pc at %v", *pc))
		}
		*pc++
		return nil
	},
	DECP: func(pc, dp, ic *int, context *Context) *byte {
		*dp--
		if *dp < 0 {
			panic(fmt.Sprintf("brainfuck: invalid data pointer, pc at %v", *pc))
		}
		*pc++
		return nil
	},
	INC: func(pc, dp, ic *int, context *Context) *byte {
		context.data[*dp]++
		*pc++
		return nil
	},
	DEC: func(pc, dp, ic *int, context *Context) *byte {
		context.data[*dp]--
		*pc++
		return nil
	},
	OUT: func(pc, dp, ic *int, context *Context) *byte {
		*pc++
		return &context.data[*dp]
	},
	IN: func(pc, dp, ic *int, context *Context) *byte {
		input_value := make([]byte, 1)
		_, err := context.input.Read(input_value)
		if err != nil {
			panic(fmt.Sprintf("brainfuck: reaching a comma but input is not available at %v", *pc))
		}
		context.data[*dp] = input_value[0]
		*pc++
		return nil
	},
	JMPF: func(pc, dp, ic *int, context *Context) *byte {
		if context.data[*dp] == 0x00 {
			*pc = context.open2Close[*pc]
		}
		*pc++
		return nil
	},
	JMPB: func(pc, dp, ic *int, context *Context) *byte {
		if context.data[*dp] != 0x00 {
			*pc = context.close2Open[*pc]
		}
		*pc++
		return nil
	},
}

type Context struct {
	// code parentheses cache
	open2Close map[int]int
	close2Open map[int]int

	data  []byte
	input io.Reader
}

func Run(codeStr string, input io.Reader) (output []byte) {
	// sanitize
	var code = []OpCode{}
	for _, v := range codeStr {
		if _, ok := CharacterToOpCode[v]; ok {
			code = append(code, CharacterToOpCode[v])
		}
	}

	// check validity of parentheses and build its cache
	type pair struct {
		index  int
		opCode OpCode
	}
	stack := []pair{}
	open2Close := make(map[int]int)
	close2Open := make(map[int]int)
	for i, v := range code {
		if v == JMPF {
			stack = append(stack, pair{
				index:  i,
				opCode: v,
			})
		} else if v == JMPB {
			if len(stack) == 0 {
				panic(fmt.Sprintf("brainfuck: invalid parenthese at %v", i))
			}
			last := len(stack) - 1
			open2Close[stack[last].index] = i
			close2Open[i] = stack[last].index
			stack = stack[0:last]
		}
	}
	if len(stack) > 0 {
		panic(fmt.Sprintf("brainfuck: invalid parenthese at %v", stack[len(stack)-1].index))
	}

	var (
		pc      int     // program counter
		dp      int     // data pointer
		ic      int     // input counter
		context Context = Context{
			open2Close: open2Close,
			close2Open: close2Open,
			data:       make([]byte, 30000),
			input:      input,
		}
	)

	// interprete loop
	for pc < len(code) {
		opCode := code[pc]
		exec := jumpTable[opCode]
		out := exec(&pc, &dp, &ic, &context)
		if out != nil {
			output = append(output, *out)
		}
	}

	return output
}
