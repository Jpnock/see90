package c90

import (
	"fmt"
	"os"
	"strings"
)

func genIndent(indent int) string {
	return strings.Repeat(" ", indent)
}

type Node interface {
	Describe(indent int) string
}

type ASTTranslationUnit []Node

func (t ASTTranslationUnit) Describe(indent int) string {
	var sb strings.Builder
	for i, node := range t {
		if i != 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(node.Describe(indent))
	}
	return sb.String()
}

type ASTBrackets struct {
	Node
}

func (t ASTBrackets) Describe(indent int) string {
	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(t.Node.Describe(0))
	sb.WriteString(")")
	return sb.String()
}

type ASTDeclarationStatementLists struct {
	decls ASTDeclaratorList
	stmts ASTStatementList
}

func (t ASTDeclarationStatementLists) Describe(indent int) string {
	var sb strings.Builder
	if t.decls != nil {
		sb.WriteString(t.decls.Describe(indent))
		sb.WriteString("\n")
	}
	if t.stmts != nil {
		sb.WriteString(t.stmts.Describe(indent))
	}
	return sb.String()
}

type ASTStatementList []Node

func (t ASTStatementList) Describe(indent int) string {
	var sb strings.Builder
	for i, decl := range t {
		if decl == nil {
			fmt.Fprintf(os.Stderr, "Found nil decl in statement list\n")
			continue
		}
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(decl.Describe(indent))
	}
	return sb.String()
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

type ASTAssignmentOperator string

const (
	ASTAssignmentOperatorEquals      ASTAssignmentOperator = "="
	ASTAssignmentOperatorMulEquals   ASTAssignmentOperator = "*="
	ASTAssignmentOperatorDivEquals   ASTAssignmentOperator = "/="
	ASTAssignmentOperatorModEquals   ASTAssignmentOperator = "%="
	ASTAssignmentOperatorAddEquals   ASTAssignmentOperator = "+="
	ASTAssignmentOperatorSubEquals   ASTAssignmentOperator = "-="
	ASTAssignmentOperatorLeftEquals  ASTAssignmentOperator = "<<="
	ASTAssignmentOperatorRightEquals ASTAssignmentOperator = ">>="
	ASTAssignmentOperatorAndEquals   ASTAssignmentOperator = "&="
	ASTAssignmentOperatorXorEquals   ASTAssignmentOperator = "^="
	ASTAssignmentOperatorOrEquals    ASTAssignmentOperator = "|="
)

type ASTIdentifier struct {
	ident string
}

func (t *ASTIdentifier) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s", genIndent(indent), t.ident)
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

type ASTAssignment struct {
	ident    string
	operator ASTAssignmentOperator
	value    Node

	// tmpAssign is set if the assignment is implicit (e.g. to a temporary
	// variable to store the resut of a comparsion etc.). In this case
	// only `value` will be set.
	tmpAssign bool
}

func (t *ASTAssignment) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.tmpAssign {
		return fmt.Sprintf("%s%s", genIndent(indent), t.value.Describe(0))
	}
	return fmt.Sprintf("%s%s %s %s", genIndent(indent), t.ident, t.operator, t.value.Describe(0))
}

type ASTArgumentExpressionList []*ASTAssignment

func (t ASTArgumentExpressionList) Describe(indent int) string {
	var sb strings.Builder
	for i, decl := range t {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(decl.Describe(indent))
	}
	return sb.String()
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

type ASTStringLiteral struct {
	value string
}

func (t *ASTStringLiteral) Describe(indent int) string {
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
	decl *ASTDirectDeclarator
	body Node
}

func (t *ASTFunction) Describe(indent int) string {
	if t == nil {
		panic("ASTFunction is nil")
	}

	indentStr := genIndent(indent)
	if t.body == nil {
		return fmt.Sprintf("%sfunction (%s) -> %s {}", indentStr, t.decl.Describe(0), t.typ.Describe(0))
	} else {
		return fmt.Sprintf("%sfunction (%s) -> %s {\n%s\n}", indentStr, t.decl.Describe(0), t.typ.Describe(0), t.body.Describe(indent+4))
	}
}

type ASTParameterList struct {
	li      []*ASTParameterDeclaration
	elipsis bool
}

func (t ASTParameterList) Describe(indent int) string {
	var sb strings.Builder
	for i, decl := range t.li {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(decl.Describe(indent))
	}
	return sb.String()
}

type ASTParameterDeclaration struct {
	specifier  Node
	declarator Node
}

func (t ASTParameterDeclaration) Describe(indent int) string {
	var sb strings.Builder
	sb.WriteString(t.specifier.Describe(indent))
	if t.declarator != nil {
		sb.WriteString(" ")
		sb.WriteString(t.declarator.Describe(0))
	}
	return sb.String()
}

type ASTDirectDeclarator struct {
	identifier *ASTIdentifier
	decl       *ASTDirectDeclarator

	// parameters is nil if it's not a function, else
	// it has zero or more parameters.
	parameters *ASTParameterList
}

func (t ASTDirectDeclarator) Describe(indent int) string {
	var sb strings.Builder

	if t.decl != nil {
		sb.WriteString(t.decl.Describe(0))
		if t.parameters != nil {
			sb.WriteString("(")
			sb.WriteString(t.parameters.Describe(0))
			sb.WriteString(")")
		}
	} else {
		sb.WriteString(t.identifier.Describe(0))
	}
	return sb.String()
}
