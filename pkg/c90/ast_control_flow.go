package c90

import (
	"fmt"
	"io"
	"strings"
)

type ASTWhileLoop struct {
	condition Node
	body      Node
}

func (t *ASTWhileLoop) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	indentStr := genIndent(indent)
	sb.WriteString(fmt.Sprintf("%swhile (%s) {", indentStr, t.condition.Describe(0)))
	if t.body != nil {
		sb.WriteString("\n")
		sb.WriteString(t.body.Describe(indent + 4))
		sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	} else {
		sb.WriteString("}")
	}
	return sb.String()
}

func (t *ASTWhileLoop) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: fix this so it work
	t.condition.GenerateMIPS(w, m)
	t.body.GenerateMIPS(w, m)
}

type ASTDoWhileLoop struct {
	condition Node
	body      Node
}

func (t *ASTDoWhileLoop) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	indentStr := genIndent(indent)
	if t.body != nil {
		sb.WriteString(fmt.Sprintf("%sdo {\n%s\n%s} ", indentStr, t.body.Describe(indent+4), indentStr))
	} else {
		sb.WriteString(fmt.Sprintf("%sdo {} ", indentStr))
	}
	sb.WriteString(fmt.Sprintf("while (%s);", t.condition.Describe(0)))
	return sb.String()
}

func (t *ASTDoWhileLoop) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTForLoop struct {
	initialiser       Node
	condition         Node
	postIterationExpr Node
	body              Node
}

func (t *ASTForLoop) Describe(indent int) string {
	if t == nil {
		return ""
	}

	indentStr := genIndent(indent)

	var sb strings.Builder
	if t.postIterationExpr == nil {
		sb.WriteString(fmt.Sprintf("%sfor(%s; %s) {", indentStr, t.initialiser.Describe(0), t.condition.Describe(0)))
	} else {
		sb.WriteString(fmt.Sprintf("%sfor(%s; %s; %s) {", indentStr, t.initialiser.Describe(0), t.condition.Describe(0), t.postIterationExpr.Describe(0)))
	}
	if t.body != nil {
		sb.WriteString("\n")
		sb.WriteString(t.body.Describe(indent + 4))
		sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	} else {
		sb.WriteString("}")
	}

	return sb.String()
}

func (t *ASTForLoop) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTIfStatement struct {
	condition Node
	body      Node
	elseBody  Node
}

func (t *ASTIfStatement) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	indentStr := genIndent(indent)
	sb.WriteString(fmt.Sprintf("%sif (%s) {", indentStr, t.condition.Describe(0)))
	if t.body != nil {
		sb.WriteString("\n")
		sb.WriteString(t.body.Describe(indent + 4))
		sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	} else {
		sb.WriteString("}")
	}

	if t.elseBody != nil {
		sb.WriteString(" else {\n")
		sb.WriteString(t.elseBody.Describe(indent + 4))
		sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	}

	return sb.String()
}

func (t *ASTIfStatement) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTReturn struct {
	returnVal Node
}

func (t *ASTReturn) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.returnVal == nil {
		return fmt.Sprintf("%sreturn (void)", genIndent(indent))
	}
	return fmt.Sprintf("%sreturn %s", genIndent(indent), t.returnVal.Describe(0))
}

func (t *ASTReturn) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTContinue struct{}

func (t *ASTContinue) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%scontinue;", genIndent(indent))
}

func (t *ASTContinue) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTBreak struct{}

func (t *ASTBreak) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%sbreak;", genIndent(indent))
}

// TODO: investigate at later date
func (t *ASTBreak) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTGoto struct {
	label *ASTIdentifier
}

func (t *ASTGoto) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%sgoto :%s;", genIndent(indent), t.label.Describe(0))
}

// TODO: investigate at later date
func (t *ASTGoto) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTLabeledStatement struct {
	ident *ASTIdentifier
	stmt  Node
}

func (t *ASTLabeledStatement) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(t.ident.Describe(0))
	sb.WriteString(":\n")
	sb.WriteString(t.stmt.Describe(indent))
	return sb.String()
}

// TODO: investigate at later date
func (t *ASTLabeledStatement) GenerateMIPS(w io.Writer, m *MIPS) {}
