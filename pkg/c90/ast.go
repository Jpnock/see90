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

type ASTDeclList []*ASTDecl

func (t ASTDeclList) Describe(indent int) string {
	var sb strings.Builder
	for i, decl := range t {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(decl.Describe(indent))
	}
	return sb.String()
}

type ASTDecl struct {
	ident string
	typ   *ASTType
}

func (t *ASTDecl) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s : %s", genIndent(indent), t.ident, t.typ.Describe(indent))
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
