package c90

import (
	"bytes"
	"fmt"
	"io"
	"os"
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

func (t *ASTFunction) Describe(indent int) string {
	if t == nil {
		panic("ASTFunction is nil")
	}

	indentStr := genIndent(indent)

	declDescribe := t.decl.Describe(0)
	funcName := declDescribe[:strings.Index(declDescribe, "(")]

	if t.body == nil {
		return fmt.Sprintf("%sfunction (%s) -> %s {}", indentStr, declDescribe, t.typ.Describe(0))
	} else {
		val := fmt.Sprintf("%sfunction (%s) -> %s {\n%s\n}\n", indentStr, declDescribe, t.typ.Describe(0), t.body.Describe(indent+4))

		buf := new(bytes.Buffer)
		buf.WriteString(fmt.Sprintf("%s:\n", funcName))

		m := NewMIPS()
		t.GenerateMIPS(buf, m)

		for _, scope := range m.VariableScopes {
			val += fmt.Sprintf("%snew scope!\n", indentStr)
			for ident, variable := range scope {
				val += fmt.Sprintf("%s%s: %v\n", indentStr, ident, *variable)
			}
		}

		fmt.Fprintf(os.Stdout, "\n\n%s", buf.String())
		return val
	}
}

func (t *ASTFunction) GenerateMIPS(w io.Writer, m *MIPS) {
	m.NewFunction()
	defer m.EndFunction()

	for i, param := range t.decl.parameters.li {
		stackOffset := 8 * (i + 1)

		// TODO: at the moment, we're assuming all function parameters are
		// identifiers, however this is clearly not the case when you have array
		// parameters.
		directDecl, ok := param.declarator.(*ASTDirectDeclarator)
		if ok {
			m.VariableScopes[len(m.VariableScopes)-1][directDecl.identifier.ident] = &Variable{
				fpOffset: -stackOffset,
				decl:     nil,
			}
		}
	}

	t.decl.GenerateMIPS(w, m)
	t.body.GenerateMIPS(w, m)
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
		sb.WriteString(arg.Describe(0))
		if i != 0 {
			sb.WriteString(", ")
		}
	}
	return fmt.Sprintf("%s%s(%s)", genIndent(indent), t.function.Describe(0), sb.String())
}

func (t *ASTFunctionCall) GenerateMIPS(w io.Writer, m *MIPS) {}
