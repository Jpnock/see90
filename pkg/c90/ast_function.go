package c90

import (
	"fmt"
	"io"
	"strings"
)

// TODO: the following is our own ABI. This needs to be changed to support MIPS
// o32 ABI.
//
// new sp
//
// new frame pointer <- return addr
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]
// 0xfffe0000 <- sp
//
// var2
// var1
// old fp
// 0xffff0000 <- fp (return addr)
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]
//
// // GenerateMips -> Function
// // string = body.GenerateMips
// // inspect the context
// // fetch the last offset used
// // subtract last offset + 12 (for ra and old fp, sp) from $sp at start of the function
//
// push 3
// push 2
// push 1
// call function
type ASTFunction struct {
	typ  *ASTType
	decl *ASTDirectDeclarator
	body Node
}

func (t *ASTFunction) Name() string {
	declDescribe := t.decl.Describe(0)
	return declDescribe[:strings.Index(declDescribe, "(")]
}

func (t *ASTFunction) Describe(indent int) string {
	if t == nil {
		panic("ASTFunction is nil")
	}

	indentStr := genIndent(indent)

	declDescribe := t.decl.Describe(0)

	if t.body == nil {
		// TODO: we'd need to generate MIPS for this but just return instantly.
		return fmt.Sprintf("%sfunction (%s) -> %s {}", indentStr, declDescribe, t.typ.Describe(0))
	}
	return fmt.Sprintf("%sfunction (%s) -> %s {\n%s\n}\n", indentStr, declDescribe, t.typ.Describe(0), t.body.Describe(indent+4))
}

func (t *ASTFunction) GenerateMIPS(w io.Writer, m *MIPS) {
	m.NewFunction()
	defer m.EndFunction()

	defer func() {
		// print the lables for strings declared in function
		write(w, ".data")
		for k, v := range m.stringMap {
			write(w, "%s:", k)
			write(w, ".asciz %s", v)
		}
		write(w, ".text")
	}()

	// Always return at end of function
	defer write(w, "jr $ra\n")

	returnLabel := m.ReturnScopes.Peek()

	funcName := t.Name()
	write(w, ".globl %s\n", funcName)
	write(w, "%s:\n", funcName)

	var variables []*Variable
	for i, param := range t.decl.parameters.li {
		stackOffset := 4 * i

		// TODO: at the moment, we're assuming all function parameters are
		// identifiers, however this is clearly not the case when you have array
		// parameters.
		directDecl, ok := param.declarator.(*ASTDirectDeclarator)
		if ok {
			v := &Variable{
				fpOffset: -stackOffset,
				decl:     nil,
				typ:      *param.specifier.(*ASTType),
			}
			m.VariableScopes[len(m.VariableScopes)-1][directDecl.identifier.ident] = v
			variables = append(variables, v)
		}
	}

	// TODO: do we need to generate mips
	// t.decl.GenerateMIPS(w, m)

	// Store $sp
	write(w, "move $t7, $sp")

	// Store $fp
	stackPush(w, "$fp", 4)
	defer stackPop(w, "$fp", 4)

	// Move frame pointer to bottom of frame
	// TODO: not ABI compliant
	write(w, "move $fp, $t7")

	// TODO: use correct length
	reserve := 8 * 20
	write(w, "addiu $sp, $sp, %d", -reserve)
	defer write(w, "addiu $sp, $sp, %d", reserve)

	for i, param := range variables {
		// TODO: change size with type
		write(w, "sw $%d, %d($fp)", i+4, -param.fpOffset)
		if i == 3 {
			break
		}
	}

	t.body.GenerateMIPS(w, m)

	write(w, "%s:", *returnLabel)
}

type ASTFunctionCall struct {
	// primary_expresion node
	function  Node
	arguments ASTArgumentExpressionList
}

func (t *ASTFunctionCall) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	for i, arg := range t.arguments {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.Describe(0))
	}
	return fmt.Sprintf("%s%s(%s)", genIndent(indent), t.function.Describe(0), sb.String())
}

func (t *ASTFunctionCall) GenerateMIPS(w io.Writer, m *MIPS) {
	stackPush(w, "$ra", 4)
	defer stackPop(w, "$ra", 4)

	numStackPop := 16
	// TODO: decide when to switch to stack based on 4x4 byte arguments
	for i, arg := range t.arguments {
		arg.GenerateMIPS(w, m)
		// TODO: switch on type of arg
		if i < 4 {
			// Arguments _definitely_ on stack after this
			write(w, "move $%d, $v0", 4+i)
		} else {
			// TODO: Sizing is wrong (and probably argument ordering)
			numStackPop += 4
			write(w, "addiu $sp, $sp, -4")
			write(w, "sw $v0, 0($sp)")
		}
	}

	write(w, "addiu $sp, $sp, -16")
	// TODO: handle arguments
	funcName := t.FunctionName()

	write(w, "jal %s", funcName)

	if numStackPop > 0 {
		write(w, "addiu $sp, $sp, %d", numStackPop)
	}
}

func (t *ASTFunctionCall) FunctionName() string {
	return t.function.Describe(0)
}
