package c90

import (
	"fmt"
	"strings"
)

func genIndent(indent int) string {
	return strings.Repeat(" ", indent)
}

type Node interface {
	Describe(indent int) string
}

type ASTDeclaratorList []*ASTDecl

func (t ASTDeclaratorList) Describe(indent int) string {
	var sb strings.Builder
	for i, decl := range t {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(decl.Describe(indent))
	}
	return sb.String()
}

type ASTIdentifier struct {
	ident string
}

func (t *ASTIdentifier) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s", genIndent(indent), t.ident)
}

type ASTDecl struct {
	ident   string
	typ     *ASTType
	initVal Node
}

func (t *ASTDecl) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.initVal == nil {
		return fmt.Sprintf("%s%s : %s", genIndent(indent), t.ident, t.typ.Describe(0))
	} else {
		return fmt.Sprintf("%s%s = %s : %s", genIndent(indent), t.ident, t.initVal.Describe(0), t.typ.Describe(0))
	}
}

type ASTConstant struct {
	value string
}

func (t *ASTConstant) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s", genIndent(indent), t.value)
}

type ASTNode struct {
	inner Node
}

func (t *ASTNode) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return t.inner.Describe(indent)
}

type ASTPanic struct{}

func (t ASTPanic) Describe(indent int) string {
	return "[panic]"
}

type ASTType struct {
	typ string
}

func (t *ASTType) Describe(indent int) string {
	if t == nil {
		panic("ASTType is nil")
	}
	return t.typ
}

type ASTFunction struct {
	typ  *ASTType
	name string
	body Node
}

func (t *ASTFunction) Describe(indent int) string {
	if t == nil {
		panic("ASTFunction is nil")
	}

	if t.body == nil {
		return fmt.Sprintf("function (%s) -> %s {}", t.name, t.typ.Describe(0))
	} else {
		return fmt.Sprintf("function (%s) -> %s {\n%s\n}", t.name, t.typ.Describe(0), t.body.Describe(indent+4))
	}
}
