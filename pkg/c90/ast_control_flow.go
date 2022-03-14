package c90

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

func checkFloatOrDoubleCondition(w io.Writer, m *MIPS) {
	falseLabel := m.CreateUniqueLabel("condtion_false")
	finalLabel := m.CreateUniqueLabel("logical_final")

	switch *m.LastType {
	case VarTypeFloat:
		write(w, "li.s $f10, 0")
		write(w, "c.eq.s $f10, $f0")
	case VarTypeDouble:
		write(w, "li.d $f10, 0")
		write(w, "c.eq.d $f10, $f0")
	default:
		return
	}

	write(w, "bc1t %s", falseLabel)

	write(w, "addiu $v0, $zero, 1")
	write(w, "j %s", finalLabel)

	write(w, "%s:", falseLabel)
	write(w, "addiu $v0, $zero, 0")

	write(w, "%s:", finalLabel)
}

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
	conditionLabel := m.CreateUniqueLabel("while_condition")
	bottomLabel := m.CreateUniqueLabel("while_bottom")

	// Create a new variable scope
	m.NewVariableScope()
	defer m.VariableScopes.Pop()

	// Create a new label scope
	m.NewLabelScope(LabelScope{
		ContinueLabel: &conditionLabel,
		BreakLabel:    &bottomLabel,
	})
	defer m.LabelScopes.Pop()

	// Condition
	write(w, "%s:", conditionLabel)
	t.condition.GenerateMIPS(w, m)
	checkFloatOrDoubleCondition(w, m)
	write(w, "beq $zero, $v0, %s", bottomLabel)

	// Body
	if t.body != nil {
		t.body.GenerateMIPS(w, m)
	}

	write(w, "j %s", conditionLabel)

	// Break to here
	write(w, "%s:", bottomLabel)
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

func (t *ASTDoWhileLoop) GenerateMIPS(w io.Writer, m *MIPS) {
	bodyLabel := m.CreateUniqueLabel("do_while_body")
	conditionLabel := m.CreateUniqueLabel("do_while_condition")
	bottomLabel := m.CreateUniqueLabel("do_while_bottom")

	// Create a new variable scope
	m.NewVariableScope()
	defer m.VariableScopes.Pop()

	// Create a new label scope
	m.NewLabelScope(LabelScope{
		ContinueLabel: &conditionLabel,
		BreakLabel:    &bottomLabel,
	})
	defer m.LabelScopes.Pop()

	// Body
	write(w, "%s:", bodyLabel)
	if t.body != nil {
		t.body.GenerateMIPS(w, m)
	}

	// Condition
	write(w, "%s:", conditionLabel)
	t.condition.GenerateMIPS(w, m)

	checkFloatOrDoubleCondition(w, m)
	write(w, "beq $zero, $v0, %s", bottomLabel)

	write(w, "j %s", bodyLabel)

	// Break to here
	write(w, "%s:", bottomLabel)
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

func (t *ASTForLoop) GenerateMIPS(w io.Writer, m *MIPS) {
	conditionLabel := m.CreateUniqueLabel("for_condition")
	bodyLabel := m.CreateUniqueLabel("for_body")
	bottomLabel := m.CreateUniqueLabel("for_bottom")
	postIterExprLabel := m.CreateUniqueLabel("for_post_iter_expr")

	// Create a new variable scope
	m.NewVariableScope()
	defer m.VariableScopes.Pop()

	// Create a new label scope
	m.NewLabelScope(LabelScope{
		ContinueLabel: &postIterExprLabel,
		BreakLabel:    &bottomLabel,
	})
	defer m.LabelScopes.Pop()

	// Init
	t.initialiser.GenerateMIPS(w, m)
	write(w, "j %s", conditionLabel)

	/// Post Iter Expression (continue from here)
	write(w, "%s:", postIterExprLabel)
	if t.postIterationExpr != nil {
		t.postIterationExpr.GenerateMIPS(w, m)
	}

	// Condition
	write(w, "%s:", conditionLabel)
	t.condition.GenerateMIPS(w, m)
	checkFloatOrDoubleCondition(w, m)
	write(w, "beq $zero, $v0, %s", bottomLabel)

	// Body
	write(w, "%s:", bodyLabel)
	if t.body != nil {
		t.body.GenerateMIPS(w, m)
	}

	write(w, "j %s", postIterExprLabel)

	// Break to here
	write(w, "%s:", bottomLabel)
}

// ASTIfStatement also works for ternary statements, as long as we keep to the
// convention that the last result is always put into v0.
type ASTIfStatement struct {
	condition Node
	body      Node
	elseBody  Node
	ternary   bool
}

func (t *ASTIfStatement) Describe(indent int) string {
	if t == nil {
		return ""
	}

	// TODO: check ternary and print differently if true
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

func (t *ASTIfStatement) GenerateMIPS(w io.Writer, m *MIPS) {
	failureLabel := m.CreateUniqueLabel("condition_fail")
	finalLabel := m.CreateUniqueLabel("condition_final")

	m.NewVariableScope()
	defer m.VariableScopes.Pop()

	// Condition
	t.condition.GenerateMIPS(w, m)

	checkFloatOrDoubleCondition(w, m)

	write(w, "beq $zero, $v0, %s", failureLabel)

	// After body, jump to end (to ignore the else clause)
	if t.body != nil {
		t.body.GenerateMIPS(w, m)
	}
	write(w, "j %s", finalLabel)

	// Else...
	write(w, "%s:", failureLabel)
	if t.elseBody != nil {
		t.elseBody.GenerateMIPS(w, m)
	}

	write(w, "%s:", finalLabel)
}

type ASTSwitchCase struct {
	// caseVal is a constexpr
	caseVal     Node
	body        Node
	defaultCase bool
}

func (t *ASTSwitchCase) Describe(indent int) string {
	if t == nil {
		return ""
	}

	var sb strings.Builder
	indentStr := genIndent(indent)
	if !t.defaultCase {
		sb.WriteString(fmt.Sprintf("%scase %s:\n", indentStr, t.caseVal.Describe(0)))
	} else {
		sb.WriteString(fmt.Sprintf("%sdefault:\n", indentStr))
	}
	if t.body != nil {
		sb.WriteString(t.body.Describe(indent + 4))
	}

	sb.WriteString(fmt.Sprintf("\n%s}", indentStr))

	return sb.String()
}

func (t *ASTSwitchCase) GenerateMIPS(w io.Writer, m *MIPS) {
	// Put label onto label scope stack
	val := &CaseLabel{
		switchCase: t,
		label:      m.CreateUniqueLabel("switch_case"),
	}
	idx := len(m.CaseLabelScopes) - 1
	m.CaseLabelScopes[idx].SwitchCase = append(m.CaseLabelScopes[idx].SwitchCase, val)

	write(w, "%s:", val.label)
	if t.body != nil {
		t.body.GenerateMIPS(w, m)
	}
}

type ASTSwitchStatement struct {
	switchOn Node
	body     Node
}

func (t *ASTSwitchStatement) Describe(indent int) string {
	if t == nil {
		return ""
	}

	var sb strings.Builder
	indentStr := genIndent(indent)
	sb.WriteString(fmt.Sprintf("%sswitch (%s) {", indentStr, t.switchOn.Describe(0)))
	if t.body != nil {
		sb.WriteString("\n")
		sb.WriteString(t.body.Describe(indent + 4))
		sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	} else {
		sb.WriteString("}")
	}

	return sb.String()
}

func (t *ASTSwitchStatement) GenerateMIPS(w io.Writer, m *MIPS) {
	bottomLabel := m.NewSwitchStatement()
	defer m.EndSwitchStatement()

	// Store the body for later. We need to execute GenerateMIPS in order to
	// explore the case labels and push them to the CaseLabelStack.
	bodyBuf := new(bytes.Buffer)
	t.body.GenerateMIPS(bodyBuf, m)

	// Put value to switch on into $t2
	t.switchOn.GenerateMIPS(w, m)
	write(w, "addu $t2, $zero, $v0")

	var defaultCase *CaseLabel
	switchCases := m.CaseLabelScopes.Peek().SwitchCase
	for _, c := range switchCases {
		if c.switchCase.defaultCase {
			// Find the default case for later.
			defaultCase = c
			continue
		}

		// Get condition into $v0
		c.switchCase.caseVal.GenerateMIPS(w, m)

		// Jump to the label if the condition matches
		write(w, "beq $v0, $t2, %s", c.label)
	}

	if defaultCase != nil {
		// Jump to the default case if no condition was matched.
		write(w, "j %s", defaultCase.label)
	} else {
		// Jump to the end of the block if no default is present.
		write(w, "j %s", bottomLabel)
	}

	w.Write(bodyBuf.Bytes())
	write(w, "%s:", bottomLabel)
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

func (t *ASTReturn) GenerateMIPS(w io.Writer, m *MIPS) {
	t.returnVal.GenerateMIPS(w, m)
	write(w, "jr $ra")
}

type ASTContinue struct{}

func (t *ASTContinue) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%scontinue;", genIndent(indent))
}

func (t *ASTContinue) GenerateMIPS(w io.Writer, m *MIPS) {
	curLabelScope := m.LabelScopes.Peek()
	write(w, "j %s", *curLabelScope.ContinueLabel)
}

type ASTBreak struct{}

func (t *ASTBreak) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%sbreak;", genIndent(indent))
}

func (t *ASTBreak) GenerateMIPS(w io.Writer, m *MIPS) {
	curLabelScope := m.LabelScopes.Peek()
	write(w, "j %s", *curLabelScope.BreakLabel)
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
func (t *ASTLabeledStatement) GenerateMIPS(w io.Writer, m *MIPS) {
	t.stmt.GenerateMIPS(w, m)
}
