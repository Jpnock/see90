package c90

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Label string

type MIPS struct {
	VariableScopes VariableScopeStack
	Context        *MIPSContext
	LabelScopes    LabelScopeStack

	uniqueLabelNumber uint
}

func NewMIPS() *MIPS {
	return &MIPS{
		VariableScopes:    nil,
		Context:           &MIPSContext{},
		LabelScopes:       nil,
		uniqueLabelNumber: 0,
	}
}

// CreateUniqueLabel takes the provided name and returns a unique label, using
// this name.
func (m *MIPS) CreateUniqueLabel(name string) Label {
	label := fmt.Sprintf("__label__%s__%d__", name, m.uniqueLabelNumber)
	m.uniqueLabelNumber++
	return Label(label)
}

type Variable struct {
	// fpOffset is the amount of subtract from fp to access the variable.
	fpOffset int
	decl     *ASTDecl
}

type MIPSContext struct {
	CurrentStackFramePointerOffset int
}

// NewScopes adds a new scope to the stack and copies all of the previous
// variables into it.
func (m *MIPS) NewVariableScope() {
	// Create a new scope and copy the last scope into it
	newScope := make(VariableScope)
	top := m.VariableScopes.Peek()
	for k, v := range top {
		newScope[k] = v
	}
	m.VariableScopes.Push(newScope)
}

// NewLabelScope adds a new label scope to the stack and copies all of the
// previous variables into it.
func (m *MIPS) NewLabelScope(l LabelScope) {
	// TODO: copy the last scope into it when we have other labels
	m.LabelScopes.Push(l)
}

// NewFunction resets context variables relating to the current function being
// generated.
func (m *MIPS) NewFunction() {
	const fp = 4
	const sp = 4
	const ra = 4
	m.Context.CurrentStackFramePointerOffset = fp + sp + ra
	m.NewVariableScope()
}

func (m *MIPSContext) GetNewLocalOffset() int {
	// TODO: change this size depending on the type of variable
	m.CurrentStackFramePointerOffset += 8
	return m.CurrentStackFramePointerOffset
}

func genIndent(indent int) string {
	return strings.Repeat(" ", indent)
}

type Node interface {
	Describe(indent int) string
	GenerateMIPS(w io.Writer, m *MIPS)
}

type ASTExpression []*ASTAssignment

func (t ASTExpression) Describe(indent int) string {
	var sb strings.Builder
	sb.WriteString(genIndent(indent))
	for i, node := range t {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(node.Describe(0))
	}
	return sb.String()
}

func (t ASTExpression) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, assignment := range t {
		assignment.GenerateMIPS(w, m)
	}
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

func (t ASTTranslationUnit) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, node := range t {
		node.GenerateMIPS(w, m)
	}
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

func (t ASTBrackets) GenerateMIPS(w io.Writer, m *MIPS) {
	t.Node.GenerateMIPS(w, m)
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

func (t ASTDeclarationStatementLists) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, node := range t.decls {
		node.GenerateMIPS(w, m)
	}
	for _, node := range t.stmts {
		node.GenerateMIPS(w, m)
	}
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

func (t ASTStatementList) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, node := range t {
		node.GenerateMIPS(w, m)
	}
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

func (t ASTDeclaratorList) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, decl := range t {
		decl.GenerateMIPS(w, m)
	}
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

func (t *ASTIdentifier) GenerateMIPS(w io.Writer, m *MIPS) {
	if t == nil {
		return
	}
	// TODO: work out how to differentiate between identifiers that don't need
	// loading into v0 (e.g. just the line `a`).

	variable := m.VariableScopes.Peek()[t.ident]
	if variable == nil {
		panic(fmt.Errorf("identifier `%s` is not in scope", t.ident))
	}
	write(w, "lw $v0, %d($fp)", -variable.fpOffset)
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

// TODO: investigate at later date
func (t *ASTAssignment) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: fix
	t.value.GenerateMIPS(w, m)

	if t.tmpAssign {
		return
	}

	assignedVar := m.VariableScopes[len(m.VariableScopes)-1][t.ident]
	write(w, "sw $v0, %d($fp)", -assignedVar.fpOffset)
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

func (t ASTArgumentExpressionList) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, decl := range t {
		decl.GenerateMIPS(w, m)
	}
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

// TODO: investigate at later date
func (t *ASTDecl) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: handle global scope case where the decl is not on the stack
	declVar := &Variable{
		fpOffset: m.Context.GetNewLocalOffset(),
		decl:     t,
	}
	m.VariableScopes[len(m.VariableScopes)-1][t.ident] = declVar
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

// TODO: investigate at later date
func (t *ASTConstant) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: fix this to support other types etc.

	intValue, _ := strconv.Atoi(t.value)
	write(w, "li $v0, %d", intValue)
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

// TODO: investigate at later date
func (t *ASTStringLiteral) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTNode struct {
	inner Node
}

func (t *ASTNode) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return t.inner.Describe(indent)
}

func (t *ASTNode) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTPanic struct{}

func (t ASTPanic) Describe(indent int) string {
	return "[panic]"
}

// TODO: investigate at later date
func (t ASTPanic) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTType struct {
	typ string
}

func (t *ASTType) Describe(indent int) string {
	if t == nil {
		panic("ASTType is nil")
	}
	return t.typ
}

// TODO: investigate at later date
func (t *ASTType) GenerateMIPS(w io.Writer, m *MIPS) {}

// new sp

// new frame pointer <- return addr
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]
// 0xfffe0000 <- sp

// var2
// var1
// old fp
// 0xffff0000 <- fp (return addr)
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]

// // GenerateMips -> Function
// // string = body.GenerateMips
// // inspect the context
// // fetch the last offset used
// // subtract last offset + 12 (for ra and old fp, sp) from $sp at start of the function

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

func (t ASTParameterList) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: this function is incomplete
	// for _, decl := range t.li {
	// 	decl.GenerateMIPS(w, m)
	// }
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

func (t *ASTParameterDeclaration) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: this function is incomplete
	//t.declarator.GenerateMIPS(w, m)
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

func (t ASTDirectDeclarator) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: fix for other fields

	// TODO: remove the function name from the variable scope
	// ident, ok :=
	// if ok {
	// 	m.VariableScopes[len(m.VariableScopes)-1][t.identifier] =
	// }

	// if t.parameters != nil {
	// 	t.parameters.GenerateMIPS(w, m)
	// }
}
