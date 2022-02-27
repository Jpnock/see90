package c90

import (
	"fmt"
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

type ASTContinue struct{}

func (t *ASTContinue) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%scontinue;", genIndent(indent))
}

type ASTBreak struct{}

func (t *ASTBreak) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%sbreak;", genIndent(indent))
}

type ASTGoto struct {
	label *ASTIdentifier
}

func (t *ASTGoto) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%sgoto :%s;", genIndent(indent), t.label.Describe(0))
}

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
