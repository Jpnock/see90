package c90

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

type VarType string

const (
	VarTypeInvalid  VarType = ""
	VarTypeInteger  VarType = "int"
	VarTypeLong     VarType = "long"
	VarTypeShort    VarType = "short"
	VarTypeFloat    VarType = "float"
	VarTypeDouble   VarType = "double"
	VarTypeChar     VarType = "char"
	VarTypeVoid     VarType = "void"
	VarTypeSigned   VarType = "signed"
	VarTypeUnsigned VarType = "unsigned"
	VarTypeString   VarType = "string"
	VarTypeTypeName VarType = "typename"
)

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

	currentlyInGlobalScope := len(m.VariableScopes) == 1
	if currentlyInGlobalScope {
		return
	}

	variable := m.VariableScopes.Peek()[t.ident]
	if variable == nil {
		panic(fmt.Errorf("identifier `%s` is not in scope", t.ident))
	}

	var globalLabel Label
	if variable.isGlobal {
		globalLabel = variable.GlobalLabel()

		// Load the address of the global into $v1
		write(w, "lui $v1, %%hi(%s)", globalLabel)
		write(w, "addiu $v1, $v1, %%lo(%s)", globalLabel)
	} else {
		// Put the address of the local into $v1
		write(w, "addiu $v1, $fp, %d", -variable.fpOffset)
	}

	m.LastType = variable.typ.typ

	switch m.LastType {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
		if variable.isGlobal {
			write(w, "lw $v0, 0($v1)")
		} else {
			write(w, "lw $v0, %d($fp)", -variable.fpOffset)
		}
	case VarTypeChar:
		if variable.isGlobal {
			write(w, "lb $v0, 0($v1)")
		} else {
			write(w, "lb $v0, %d($fp)", -variable.fpOffset)
		}
	case VarTypeFloat:
		if variable.isGlobal {
			write(w, "lwc1 $f0, 0($v1)")
		} else {
			write(w, "lwc1 $f0, %d($fp)", -variable.fpOffset)
		}
	case VarTypeDouble:
		if variable.isGlobal {
			write(w, "lwc1 $f0, 4($v1)")
			write(w, "lwc1 $f1, 0($v1)")
		} else {
			write(w, "lwc1 $f0, %d($fp)", -variable.fpOffset+4)
			write(w, "lwc1 $f1, %d($fp)", -variable.fpOffset)
		}
	case VarTypeString:
		write(w, "lui $v0, %%hi(%s)", *variable.label)
		write(w, "addiu $v0, $v0, %%lo(%s)", *variable.label)
	default:
		panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
	}
}

type ASTAssignment struct {
	lval     Node
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
	return fmt.Sprintf("%s%s %s %s", genIndent(indent), t.lval.Describe(0), t.operator, t.value.Describe(0))
}

func storeToReturnRegister(w io.Writer, typ VarType) {
	switch typ {
	case VarTypeFloat:
		write(w, "swc1 $f0, 0($v1)")
	case VarTypeDouble:
		write(w, "swc1 $f0, 4($v1)")
		write(w, "swc1 $f1, 0($v1)")
	default:
		write(w, "sw $v0, 0($v1)")
	}
	return
}

// TODO: investigate at later date
func (t *ASTAssignment) GenerateMIPS(w io.Writer, m *MIPS) {
	// Load value into $v0/$f0
	t.value.GenerateMIPS(w, m)

	if t.tmpAssign || m.LastType == VarTypeString {
		return
	}

	rhsType := m.LastType

	// TODO: switch on type
	switch rhsType {
	case VarTypeFloat:
		stackPushFP(w, "$f0")
		t.lval.GenerateMIPS(w, m)
		stackPopFP(w, "$f0")
	case VarTypeDouble:
		stackPushFP(w, "$f0", "$f1")
		t.lval.GenerateMIPS(w, m)
		stackPopFP(w, "$f0", "$f1")
	default:
		stackPush(w, "$v0", 4)
		t.lval.GenerateMIPS(w, m)
		stackPop(w, "$v0", 4)
	}

	if t.operator == ASTAssignmentOperatorEquals {
		// Special case as this does not require a load
		storeToReturnRegister(w, m.LastType)
		return
	}

	switch m.LastType {
	case VarTypeFloat:
		write(w, "lwc1 $f2, 0($v1)")
	case VarTypeDouble:
		write(w, "lwc1 $f2, 4($v1)")
		write(w, "lwc1 $f3, 0($v1)")
	default:
		write(w, "lw $t0, 0($v1)")
	}

	switch t.operator {
	case ASTAssignmentOperatorMulEquals:
		switch rhsType {
		case VarTypeFloat:
			write(w, "mul.s $f0, $f2, $f0")
		case VarTypeDouble:
			write(w, "mul.d $f0, $f2, $f0")
		case VarTypeUnsigned:
			write(w, "multu $t0, $v0")
			write(w, "mflo $v0")
		default:
			write(w, "mult $t0, $v0")
			write(w, "mflo $v0")
		}
	case ASTAssignmentOperatorDivEquals:
		switch rhsType {
		case VarTypeFloat:
			write(w, "div.s $f0, $f2, $f0")
		case VarTypeDouble:
			write(w, "div.d $f0, $f2, $f0")
		case VarTypeUnsigned:
			write(w, "divu $t0, $v0")
			write(w, "mflo $v0")
		default:
			write(w, "div $t0, $v0")
			write(w, "mflo $v0")
		}
	case ASTAssignmentOperatorAddEquals:
		switch rhsType {
		case VarTypeFloat:
			write(w, "add.s $f0, $f2, $f0")
		case VarTypeDouble:
			write(w, "add.d $f0, $f2, $f0")
		default:
			write(w, "addu $v0, $t0, $v0")
		}
	case ASTAssignmentOperatorSubEquals:
		switch rhsType {
		case VarTypeFloat:
			write(w, "sub.s $f0, $f2, $f0")
		case VarTypeDouble:
			write(w, "sub.d $f0, $f2, $f0")
		default:
			write(w, "subu $v0, $t0, $v0")
		}
	case ASTAssignmentOperatorModEquals:
		write(w, "div $t0, $v0")
		write(w, "mfhi $v0")
	case ASTAssignmentOperatorLeftEquals:
		write(w, "sllv $v0, $t0, $v0")
	case ASTAssignmentOperatorRightEquals:
		write(w, "srlv $v0, $t0, $v0")
	case ASTAssignmentOperatorAndEquals:
		write(w, "and $v0, $t0, $v0")
	case ASTAssignmentOperatorXorEquals:
		write(w, "xor $v0, $t0, $v0")
	case ASTAssignmentOperatorOrEquals:
		write(w, "or $v0, $t0, $v0")
	default:
		panic("unhanlded ASTAssignmentOperator")
	}

	storeToReturnRegister(w, m.LastType)
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
	decl    *ASTDirectDeclarator
	typ     *ASTType
	initVal Node
}

func (t *ASTDecl) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.initVal == nil {
		return fmt.Sprintf("%s%s : %s", genIndent(indent), t.decl.Describe(0), t.typ.Describe(0))
	} else {
		return fmt.Sprintf("%s%s = %s : %s", genIndent(indent), t.decl.Describe(0), t.initVal.Describe(0), t.typ.Describe(0))
	}
}

// TODO: investigate at later date
func (t *ASTDecl) GenerateMIPS(w io.Writer, m *MIPS) {
	if t.decl == nil || t.decl.identifier == nil {
		// TODO: handle this case (mostly caused by function prototypes).
		return
	}

	isGlobal := len(m.VariableScopes) == 1
	declVar := &Variable{
		decl:     t,
		typ:      *t.typ,
		label:    nil,
		isGlobal: isGlobal,
	}

	var globalLabel Label
	if !isGlobal {
		declVar.fpOffset = m.Context.GetNewLocalOffset()
	} else {
		globalLabel = declVar.GlobalLabel()
	}

	m.LastType = t.typ.typ
	m.VariableScopes[len(m.VariableScopes)-1][t.decl.identifier.ident] = declVar

	if isGlobal {
		// Global variable
		write(w, ".data")
		defer write(w, ".text")
		write(w, "%s:", globalLabel)
		if t.initVal == nil {
			switch t.typ.typ {
			case VarTypeChar:
				write(w, "  .byte 0")
			case VarTypeDouble:
				write(w, "  .word 0")
				write(w, "  .word 0")
			default:
				write(w, "  .word 0")
			}
			return
		}
		t.initVal.GenerateMIPS(w, m)
		return
	}

	// Local variable
	if t.initVal == nil {
		return
	}

	t.initVal.GenerateMIPS(w, m)

	if m.LastType == VarTypeString {
		declVar.label = &m.lastLabel
		declVar.typ = ASTType{typ: VarTypeString, typName: ""}
		m.VariableScopes[len(m.VariableScopes)-1][t.decl.identifier.ident] = declVar
		return
	}

	switch t.typ.typ {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
		write(w, "sw $v0, %d($fp)", -declVar.fpOffset)
	case VarTypeChar:
		write(w, "sb $v0, %d($fp)", -declVar.fpOffset)
	case VarTypeFloat:
		write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset)
	case VarTypeDouble:
		write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+4)
		write(w, "swc1 $f1, %d($fp)", -declVar.fpOffset)
	default:
		panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
	}

	m.VariableScopes[len(m.VariableScopes)-1][t.decl.identifier.ident] = declVar
}

type ASTConstant struct {
	value string
}

func (t *ASTConstant) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.value[0] == '\'' {
		ascii := int(([]rune(t.value))[1])
		return fmt.Sprintf("%s%d", genIndent(indent), ascii)
	}
	return fmt.Sprintf("%s%s", genIndent(indent), t.value)
}

// TODO: investigate at later date
func (t *ASTConstant) GenerateMIPS(w io.Writer, m *MIPS) {
	if len(t.value) == 0 {
		panic("empty ASTConstant")
	}

	isGlobal := len(m.VariableScopes) == 1

	// TODO: fix this to support other types etc.

	// TODO: currently doesnt detect chars declard with an int not a char literal
	if t.value[0] == '\'' {
		unquotedString, err := strconv.Unquote(t.value)
		if err != nil {
			panic(fmt.Errorf("character literal unquote gave error: %v", err))
		}
		if isGlobal {
			write(w, ".byte %d", int(unquotedString[0]))
		} else {
			write(w, "li $v0, %d", unquotedString[0])
		}
		m.LastType = VarTypeChar
		return
	}

	lastIdx := len(t.value) - 1

	// Try to parse the constant as a float (or double) and load
	// it into $f0.
	if t.value[lastIdx] == 'f' || t.value[lastIdx] == 'F' {
		// Appendix A, pg. 194 states that all numbers are doubles (or long doubles)
		// unless suffixed with f or F, which implies they are floats.
		f32, err := strconv.ParseFloat(t.value[:lastIdx], 32)
		if err != nil {
			panic("invalid floating point constant")
		}
		if isGlobal {
			write(w, "  .word %d", math.Float32bits(float32(f32)))
		} else {
			write(w, "li.s $f0, %f", float32(f32))
		}
		m.LastType = VarTypeFloat
		return
	}

	if t.value[lastIdx] == 'u' || t.value[lastIdx] == 'U' {
		// Appendix A, pg. 194 states that all numbers are doubles (or long doubles)
		// unless suffixed with f or F, which implies they are floats.
		uintValue, err := strconv.ParseUint(t.value[:lastIdx], 0, 32)
		if err != nil {
			panic("unable to convert unsinged to int")
		}
		if isGlobal {
			write(w, "  .word %d", uintValue)
		} else {
			write(w, "li $v0, %d", uintValue)
		}
		return
	}

	emittedGlobalInt := false
	intValue, err := strconv.ParseInt(t.value, 0, 32)
	if err == nil {
		// Could be an integer or double (assume integer as all operations
		// can be performed on this type; it will also be overwritten by
		// ASTDecl/ASTIdentifier etc.)
		if isGlobal {
			if m.LastType != VarTypeDouble {
				emittedGlobalInt = true
				write(w, "  .word %d", intValue)
			}
		} else {
			write(w, "li $v0, %d", intValue)
		}
		m.LastType = VarTypeInteger
	} else {
		// Not an int
		m.LastType = VarTypeDouble
	}

	f64, err := strconv.ParseFloat(t.value, 64)
	if err != nil {
		panic("ASTConstant expected double")
	}
	if isGlobal {
		if !emittedGlobalInt {
			bits := math.Float64bits(f64)
			write(w, "  .word %d", bits>>32)
			write(w, "  .word %d", bits&0xFFFFFFFF)
		}
		return
	}
	write(w, "li.d $f0, %f", f64)
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
func (t *ASTStringLiteral) GenerateMIPS(w io.Writer, m *MIPS) {

	//Get slice of escaped runes
	unquotedString, err := strconv.Unquote(t.value)
	if err != nil {
		panic(fmt.Errorf("string Literal unquote gave error: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("\"")
	//for each rune convert them into hex and add \x before hand then add that to the string
	for _, r := range unquotedString {
		sb.WriteString(
			fmt.Sprintf("\\x%02x", r),
		)
	}
	sb.WriteString("\\000\"")

	stringlabel := m.CreateUniqueLabel("string")

	m.lastLabel = stringlabel
	m.stringMap[stringlabel] = sb.String()

	m.LastType = VarTypeString
}

type ASTPanic struct{}

func (t ASTPanic) Describe(indent int) string {
	return "[panic]"
}

// TODO: investigate at later date
func (t ASTPanic) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTType struct {
	typ     VarType
	typName string
}

func (t *ASTType) Describe(indent int) string {
	if t == nil {
		panic("ASTType is nil")
	}
	return string(t.typ)
}

// TODO: investigate at later date
func (t *ASTType) GenerateMIPS(w io.Writer, m *MIPS) {
	m.LastType = t.typ
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

	isPointer bool

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

type ASTScope struct {
	body Node
}

func (t *ASTScope) Describe(indent int) string {
	if t.body == nil {
		return ""
	}
	return t.body.Describe(indent)
}

func (t *ASTScope) GenerateMIPS(w io.Writer, m *MIPS) {
	if t.body == nil {
		return
	}
	m.NewVariableScope()
	t.body.GenerateMIPS(w, m)
	m.VariableScopes.Pop()
}

func genIndent(indent int) string {
	return strings.Repeat(" ", indent)
}
