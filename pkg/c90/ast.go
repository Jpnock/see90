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
	VarTypeEnum     VarType = "enum"
	VarTypeStruct   VarType = "struct"
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

	write(w, "%s:", string(m.CreateUniqueLabel("identgen")))
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

		if variable.typ.typ == VarTypeString {
			globalLabel = *variable.label
		}
		// Load the address of the global into $v1
		write(w, "lui $v1, %%hi(%s)", globalLabel)
		write(w, "addiu $v1, $v1, %%lo(%s)", globalLabel)
	} else {
		// Put the address of the local into $v1
		write(w, "addiu $v1, $fp, %d", -variable.fpOffset)
	}

	m.SetLastType(variable.typ.typ)
	if variable.directDecl != nil {
		m.pointerLevel = variable.directDecl.pointerDepth
	}

	if variable.IsArray() {
		if variable.isGlobal {
			write(w, "lui $v0, %%hi(%s)", globalLabel)
			write(w, "addiu $v0, $v0, %%lo(%s)", globalLabel)
			return
		}
		write(w, "addiu $v0, $fp, %d", -variable.fpOffset)

		if variable.isLocalDataString {
			// TODO: make this better
			// array is in .data section, so we need to dereference.
			write(w, "lw $v0, 0($v0)")
			write(w, "lw $v1, 0($v1)")
		}
		// Arrays have the same value as their address
		return
	}

	switch m.LastType() {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeStruct:
		if variable.isGlobal {
			write(w, "lw $v0, 0($v1)")
		} else {
			write(w, "lw $v0, %d($fp)", -variable.fpOffset)
		}
	case VarTypeEnum:
		variable.enum.value.GenerateMIPS(w, m)
		if variable.enum.offset != 0 {
			write(w, "addiu $v0, $v0, %d", variable.enum.offset)
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
	case VarTypeChar:
		write(w, "sb $v0, 0($v1)")
	default:
		write(w, "sw $v0, 0($v1)")
	}
}

// TODO: investigate at later date
func (t *ASTAssignment) GenerateMIPS(w io.Writer, m *MIPS) {
	// Load value into $v0/$f0
	t.value.GenerateMIPS(w, m)

	if t.tmpAssign || m.LastType() == VarTypeString {
		return
	}

	rhsType := m.LastType()

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
		storeToReturnRegister(w, m.LastType())
		return
	}

	switch m.LastType() {
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

	storeToReturnRegister(w, m.LastType())
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

type ASTInitializerList []Node

func (t ASTInitializerList) Describe(indent int) string {
	var sb strings.Builder
	sb.WriteString("{")
	for i, decl := range t {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(decl.Describe(indent))
	}
	sb.WriteString("}")
	return sb.String()
}

func (t ASTInitializerList) GenerateMIPS(w io.Writer, m *MIPS) {
	// for _, decl := range t {
	// 	decl.GenerateMIPS(w, m)
	// }
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

	if t.decl == nil && t.typ != nil && (t.typ.typ == VarTypeEnum || t.typ.typ == VarTypeStruct) {
		return fmt.Sprintf("%s;", t.typ.Describe(indent))
	}

	pointers := ""
	if t.decl != nil && t.decl.pointerDepth > 0 {
		pointers = strings.Repeat("*", t.decl.pointerDepth)
	}

	if t.initVal == nil {
		return fmt.Sprintf("%s%s : %s%s", genIndent(indent), t.decl.Describe(0), pointers, t.typ.Describe(0))
	} else {
		if t.typ != nil && t.typ.typ == VarTypeStruct {
			return fmt.Sprintf("%s%s = { %s } : struct %s%s", genIndent(indent), t.decl.Describe(0), t.initVal.Describe(0), pointers, t.typ.Describe(0))
		}
		return fmt.Sprintf("%s%s = %s : %s%s", genIndent(indent), t.decl.Describe(0), t.initVal.Describe(0), pointers, t.typ.Describe(0))
	}
}

func (t *ASTDecl) isPointer() bool {
	return t.decl.pointerDepth > 0
}

func (t *ASTDecl) isArray() bool {
	return t.decl.array != nil
}

func (t *ASTDecl) getArrayInfo(m *MIPS) (dimensions []int, totalElements int, sizeOf int) {
	// Work out how many bytes to reserve
	dims := t.decl.ArrayDimensions()

	totalElements = 1
	if len(dims) == 0 {
		totalElements = 0
	}
	for _, dim := range dims {
		totalElements *= dim
	}

	sizeOfElement := m.sizeOfType(t.typ.typ, t.isPointer())
	reserveArrayBytes := sizeOfElement * totalElements
	return dims, totalElements, reserveArrayBytes
}

func (t *ASTDecl) generateLocalVarMIPSStruct(w io.Writer, m *MIPS, ident *ASTIdentifier, declVar *Variable) {
	structType := *m.StructScopes[len(m.StructScopes)-1][t.typ.typName]
	m.Context.GetNewLocalOffsetWithMinSize(structType.totalOffsetSize)

	// TODO: is this fine if we're here from an initializer list containing a struct?
	declVar.structure = &structType

	var numOfInitilizers int
	if t.initVal != nil {
		numOfInitilizers = len(t.initVal.(ASTInitializerList))
		for i, element := range t.initVal.(ASTInitializerList) {
			// TODO: handle nested init list

			element.GenerateMIPS(w, m)

			// TODO: handle pointer/array types
			switch structType.types[i].typ {
			case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
				write(w, "sw $v0, %d($fp)", -declVar.fpOffset+structType.offsets[i])
			case VarTypeChar:
				write(w, "sb $v0, %d($fp)", -declVar.fpOffset+structType.offsets[i])
			case VarTypeFloat:
				write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+structType.offsets[i])
			case VarTypeDouble:
				write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+structType.offsets[i]+4)
				write(w, "swc1 $f1, %d($fp)", -declVar.fpOffset+structType.offsets[i])
			default:
				panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
			}
		}
	} else {
		numOfInitilizers = 0
	}

	numOfElements := len(structType.elementIdents)
	if numOfElements > numOfInitilizers {
		for i := 0; i < (numOfInitilizers - numOfElements); i++ {
			// TODO: handle pointer/array types
			switch structType.types[numOfInitilizers+i].typ {
			case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
				write(w, "addiu $v0, $v0, 0")
				write(w, "sw $v0, %d($fp)", -declVar.fpOffset+structType.offsets[numOfInitilizers+i])
			case VarTypeChar:
				write(w, "addiu $v0, $v0, 0")
				write(w, "sb $v0, %d($fp)", -declVar.fpOffset+structType.offsets[numOfInitilizers+i])
			case VarTypeFloat:
				write(w, "li.s $f0, 0")
				write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+structType.offsets[numOfInitilizers+i])
			case VarTypeDouble:
				write(w, "li.d $f0, 0")
				write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+structType.offsets[numOfInitilizers+i]+4)
				write(w, "swc1 $f1, %d($fp)", -declVar.fpOffset+structType.offsets[numOfInitilizers+i])
			default:
				panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
			}
		}
	}
	return
}

func (t *ASTDecl) generateLocalVarMIPS(w io.Writer, m *MIPS, ident *ASTIdentifier, declVar *Variable) {
	isArray := t.decl.array != nil

	if isArray {
		_, _, reserveArrayBytes := t.getArrayInfo(m)
		declVar.fpOffset = m.Context.GetNewLocalOffsetWithMinSize(reserveArrayBytes)
	} else if t.typ.typ == VarTypeStruct {
		t.generateLocalVarMIPSStruct(w, m, ident, declVar)
		return
	} else {
		declVar.fpOffset = m.Context.GetNewLocalOffset()
	}

	if t.initVal == nil {
		return
	}

	elements := []Node{t.initVal}
	if initializerList, ok := t.initVal.(ASTInitializerList); isArray && ok {
		// Generate array with initializer list RHS instead
		elements = nil

		_, numElements, _ := t.getArrayInfo(m)
		for i, entry := range initializerList {
			if i >= numElements {
				// Not enough space in the array
				break
			}

			if _, ok := entry.(ASTInitializerList); ok {
				// TODO: handle nested entries
				panic("entry is an init list which is not yet handled")
			}

			elements = append(elements, entry)
		}
	}

	assignment, ok := t.initVal.(*ASTAssignment)
	if ok {
		if _, ok := assignment.value.(*ASTStringLiteral); ok {
			declVar.isLocalDataString = true
		}
	}

	for i, element := range elements {
		// Value is in $v0/f0, so now we just need to store it
		element.GenerateMIPS(w, m)

		switchType := t.typ.typ
		if m.LastType() == VarTypeString || (switchType == VarTypeChar && t.isPointer()) {
			switchType = VarTypeString
		}

		if len(elements) == 1 && t.isArray() && switchType == VarTypeChar {
			switchType = VarTypeUnsigned
		}

		switch switchType {
		case VarTypeChar:
			write(w, "sb $v0, %d($fp)", -declVar.fpOffset+i)
		case VarTypeFloat:
			write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+(i*4))
		case VarTypeDouble:
			write(w, "swc1 $f0, %d($fp)", -declVar.fpOffset+4+(i*4))
			write(w, "swc1 $f1, %d($fp)", -declVar.fpOffset+(i*4))
		case VarTypeString:
			if isArray {
				strBytes := m.stringMap[m.lastLabel]
				_, _, reserveArrayBytes := t.getArrayInfo(m)
				for i := 0; i < reserveArrayBytes; i++ {
					if i < len(strBytes) {
						write(w, "li $t0, %d", strBytes[i])
						write(w, "sb $t0, %d($fp)", -declVar.fpOffset+i)
					} else {
						// Null terminated. If no initialiser is provided
						// then maybe we shouldn't be doing this?
						write(w, "sb $zero, %d($fp)", -declVar.fpOffset+i)
					}
				}
			} else {
				// Generate later in the .data section
				declVar.label = &m.lastLabel
				declVar.typ = ASTType{typ: VarTypeString, typName: ""}
				m.VariableScopes[len(m.VariableScopes)-1][ident.ident] = declVar
				write(w, "sw $v0, %d($fp)", -declVar.fpOffset+(i*4))
			}
		default:
			write(w, "sw $v0, %d($fp)", -declVar.fpOffset+(i*4))
		}
	}
}

func (t *ASTDecl) generateGlobalVarMIPS(w io.Writer, m *MIPS, ident *ASTIdentifier, declVar *Variable) {
	write(w, ".data")
	defer write(w, ".text")
	write(w, "%s:", declVar.GlobalLabel())

	isArray := t.isArray()

	if t.initVal == nil {
		// Reserve space at the label, even if there is no initial value.
		if isArray {
			_, _, reserveArrayBytes := t.getArrayInfo(m)
			for i := 0; i < reserveArrayBytes; i++ {
				write(w, "    .byte 0")
			}
			return
		}
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

	elements := []Node{t.initVal}
	if initializerList, ok := t.initVal.(ASTInitializerList); isArray && ok {
		// No longer generate t.initVal
		elements = nil

		_, totalElements, _ := t.getArrayInfo(m)
		for i, entry := range initializerList {
			if i >= totalElements {
				// Not enough space in the array
				break
			}

			if _, ok := entry.(ASTInitializerList); ok {
				// TODO: handle nested entries
				panic("entry is an init list which is not yet handled")
			}

			elements = append(elements, entry)
		}
	}

	// Global initializers have to be constants
	for _, element := range elements {
		fmt.Fprintf(os.Stderr, "Got type %T\n", element)
		assignmentExpr := element.(*ASTAssignment)
		if _, ok := assignmentExpr.value.(*ASTStringLiteral); ok {
			// TODO: handle this better (for char * array as there will be
			// multiple strings to set labels for)
			declVar.label = &m.lastLabel
			declVar.typ = ASTType{typ: VarTypeString, typName: ""}
			element.GenerateMIPS(w, m)
			continue
		}

		val := EvaluateConstExpr(element)
		switch t.typ.typ {
		case VarTypeChar:
			emitGlobalChar(w, uint8(val))
		case VarTypeDouble:
			emitGlobalDouble(w, val)
		case VarTypeFloat:
			emitGlobalFloat(w, float32(val))
		case VarTypeUnsigned:
			emitGlobalUint32(w, uint32(val))
		default:
			emitGlobalInt32(w, int32(val))
		}
	}
}

// TODO: investigate at later date
func (t *ASTDecl) GenerateMIPS(w io.Writer, m *MIPS) {
	if t.decl == nil && t.typ != nil && (t.typ.typ == VarTypeEnum || t.typ.typ == VarTypeStruct) {
		t.typ.GenerateMIPS(w, m)
		return
	}

	if t.decl == nil {
		return
	}

	ident := t.decl.Identifier()
	if ident == nil {
		// TODO: handle this case (mostly caused by function prototypes).
		return
	}

	isGlobal := len(m.VariableScopes) == 1
	declVar := &Variable{
		decl:       t,
		directDecl: t.decl,
		typ:        *t.typ,
		label:      nil,
		isGlobal:   isGlobal,
	}

	m.SetLastType(t.typ.typ)
	m.VariableScopes[len(m.VariableScopes)-1][ident.ident] = declVar

	if isGlobal {
		t.generateGlobalVarMIPS(w, m, ident, declVar)
		m.pointerLevel = t.decl.pointerDepth
		return
	}

	m.pointerLevel = t.decl.pointerDepth
	t.generateLocalVarMIPS(w, m, ident, declVar)
	m.pointerLevel = t.decl.pointerDepth
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

func emitGlobalFloat(w io.Writer, val float32) {
	write(w, "  .word %d", math.Float32bits(val))
}

func emitGlobalDouble(w io.Writer, val float64) {
	bits := math.Float64bits(val)
	write(w, "  .word %d", bits>>32)
	write(w, "  .word %d", bits&0xFFFFFFFF)
}

func emitGlobalInt32(w io.Writer, val int32) {
	write(w, "  .word %d", val)
}

func emitGlobalUint32(w io.Writer, val uint32) {
	write(w, "  .word %d", val)
}

func emitGlobalChar(w io.Writer, val uint8) {
	write(w, ".byte %d", val)
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
		if t.value == `'\0'` {
			// Handle special case as this is different
			// in Go to C.
			t.value = `'\000'`
		}
		unquotedString, err := strconv.Unquote(t.value)
		if err != nil {
			panic(fmt.Errorf("character literal unquote gave error: %v", err))
		}
		if isGlobal {
			emitGlobalChar(w, uint8(unquotedString[0]))
		} else {
			write(w, "li $v0, %d", unquotedString[0])
		}
		m.SetLastType(VarTypeChar)
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
			emitGlobalFloat(w, float32(f32))
		} else {
			write(w, "li.s $f0, %f", float32(f32))
		}
		m.SetLastType(VarTypeFloat)
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
			emitGlobalUint32(w, uint32(uintValue))
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
			if m.LastType() != VarTypeDouble {
				emittedGlobalInt = true
				emitGlobalInt32(w, int32(intValue))
			}
		} else {
			write(w, "li $v0, %d", intValue)
		}
		m.SetLastType(VarTypeInteger)
	} else {
		// Not an int
		m.SetLastType(VarTypeDouble)
	}

	f64, err := strconv.ParseFloat(t.value, 64)
	if err != nil {
		panic("ASTConstant expected double")
	}
	if isGlobal {
		if !emittedGlobalInt {
			emitGlobalDouble(w, f64)
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

	stringlabel := m.CreateUniqueLabel("string")

	m.lastLabel = stringlabel
	m.stringMap[stringlabel] = []byte(unquotedString)

	isGlobal := len(m.VariableScopes) == 1
	if isGlobal {
		writeGlobalString(w, stringlabel, []byte(unquotedString))
	} else {
		write(w, "lui $v0, %%hi(%s_data)", stringlabel)
		write(w, "addiu $v0, $v0, %%lo(%s_data)", stringlabel)
	}

	m.SetLastType(VarTypeChar)
	m.pointerLevel = 1
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

	enum      *ASTEnum
	structure *ASTStruct
}

func (t *ASTType) Describe(indent int) string {
	if t == nil {
		panic("ASTType is nil")
	}

	if t.typ == VarTypeEnum {
		return t.enum.Describe(indent)
	}

	if t.typ == VarTypeStruct {
		return t.structure.Describe(indent)
	}

	if t.typ == VarTypeStruct {
		return string(t.typName)
	}

	return string(t.typ)
}

// TODO: investigate at later date
func (t *ASTType) GenerateMIPS(w io.Writer, m *MIPS) {
	switch t.typ {
	case VarTypeEnum:
		m.SetLastType(VarTypeUnsigned)
		// TODO: we might have some problems with struct parameters?
		t.enum.GenerateMIPS(w, m)
	case VarTypeStruct:
		// TODO: not sure what to set last type to
		// TODO: we might have some problems with struct parameters?
		t.structure.GenerateMIPS(w, m)
	default:
		m.SetLastType(t.typ)
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

	pointerDepth int

	// parameters is nil if it's not a function, else
	// it has zero or more parameters.
	parameters *ASTParameterList

	// Nil if the direct decl is not an array
	array *ASTArray
}

func (t ASTDirectDeclarator) ArrayDimensions() []int {
	if t.array == nil {
		return nil
	}

	dimensions := t.decl.ArrayDimensions()
	dimensions = append(dimensions, t.array.size)
	return dimensions
}

func (t ASTDirectDeclarator) Identifier() *ASTIdentifier {
	root := &t
	for root != nil {
		if root.identifier != nil {
			return root.identifier
		}
		root = root.decl
	}
	return nil
}

func (t ASTDirectDeclarator) Describe(indent int) string {
	var sb strings.Builder

	if t.decl != nil {
		sb.WriteString(t.decl.Describe(0))
		if t.array != nil {
			sb.WriteString("[")
			sb.WriteString(fmt.Sprintf("%d", t.array.size))
			sb.WriteString("]")
		}
		if t.parameters != nil {
			sb.WriteString("(")
			sb.WriteString(t.parameters.Describe(0))
			sb.WriteString(")")
		}
	} else if t.array != nil {
		sb.WriteString("[")
		sb.WriteString(fmt.Sprintf("%d", t.array.size))
		sb.WriteString("]")
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
	m.NewStructScope()
	t.body.GenerateMIPS(w, m)
	m.VariableScopes.Pop()
	m.StructScopes.Pop()
}

type ASTEnum struct {
	ident   *ASTIdentifier
	entries ASTEnumEntryList
}

func NewASTEnum(ident *ASTIdentifier, entries ASTEnumEntryList) *ASTEnum {
	enum := &ASTEnum{
		ident:   ident,
		entries: entries,
	}

	if len(entries) == 0 {
		return enum
	}

	if enum.entries[0].value == nil {
		enum.entries[0].value = &ASTConstant{value: "0"}
	}

	lastNonNilValue := enum.entries[0].value
	lastNonNilValueIndex := 0
	for i, entry := range enum.entries {
		if entry.value == nil {
			enum.entries[i].offset = i - lastNonNilValueIndex
			enum.entries[i].value = lastNonNilValue
		} else {
			lastNonNilValue = entry.value
			lastNonNilValueIndex = i
		}
	}
	return enum
}

func (t *ASTEnum) Describe(indent int) string {
	indentStr := genIndent(indent)

	var sb strings.Builder
	if t.ident != nil {
		sb.WriteString(
			fmt.Sprintf("%senum %s {", indentStr, t.ident.Describe(0)),
		)
	} else {
		sb.WriteString(
			fmt.Sprintf("%senum {", indentStr),
		)
	}
	for _, entry := range t.entries {
		sb.WriteString("\n")
		sb.WriteString(
			fmt.Sprintf("%s    %s", indentStr, entry.Describe(0)),
		)
	}
	sb.WriteString(fmt.Sprintf("\n%s}", indentStr))
	return sb.String()
}

func (t *ASTEnum) GenerateMIPS(w io.Writer, m *MIPS) {
	for _, entry := range t.entries {
		variableEntry := entry
		m.VariableScopes[len(m.VariableScopes)-1][entry.ident.ident] = &Variable{
			typ:  ASTType{typ: VarTypeEnum},
			enum: variableEntry,
		}
	}
}

type ASTEnumEntryList []*ASTEnumEntry

func (t ASTEnumEntryList) Describe(indent int) string {
	return ""
}

func (t ASTEnumEntryList) GenerateMIPS(w io.Writer, m *MIPS) {
	return
}

type ASTEnumEntry struct {
	ident  *ASTIdentifier
	value  Node
	offset int
}

func (t ASTEnumEntry) Describe(indent int) string {
	var sb strings.Builder

	sb.WriteString(t.ident.ident)
	sb.WriteString(" = (")
	sb.WriteString(t.value.Describe(indent))
	sb.WriteString(")")
	if t.offset != 0 {
		sb.WriteString(" + ")
		sb.WriteString(fmt.Sprintf("%d", t.offset))
	}

	return sb.String()
}

func (t ASTEnumEntry) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTStruct struct {
	ident    *ASTIdentifier
	elements ASTStructDeclarationList
}

func (t ASTStruct) Describe(indent int) string {
	var sb strings.Builder
	sindent := genIndent(indent)
	sb.WriteString(fmt.Sprintf("%sstruct %s {\n", sindent, t.ident.ident))
	sb.WriteString(t.elements.Describe(indent))
	sb.WriteString(fmt.Sprintf("%s}", sindent))
	return sb.String()
}

func (t ASTStruct) GenerateMIPS(w io.Writer, m *MIPS) {
	structEntry := Struct{ident: t.ident.ident, offsets: make(map[int]int), types: make(map[int]ASTType), elementIdents: make(map[string]int)}

	var structSize = 0
	var totalOffsetSize = 0
	for i, elementSlice := range t.elements {
		for j, element := range elementSlice {
			structEntry.offsets[i+j] = totalOffsetSize
			structEntry.types[i+j] = *element.decl.typ
			structEntry.elementIdents[element.decl.decl.identifier.ident] = i + j
			totalOffsetSize += 8
			switch element.decl.typ.typ {
			case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeFloat:
				structSize += 4
			case VarTypeChar:
				structSize += 1
			case VarTypeDouble:
				structSize += 8
			default:
				panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
			}

		}
	}

	structEntry.totalOffsetSize = totalOffsetSize
	structEntry.structSize = structSize
	m.StructScopes[len(m.StructScopes)-1][t.ident.ident] = &structEntry
}

type ASTStructDeclarator struct {
	decl *ASTDecl
}

func (t ASTStructDeclarator) Describe(indent int) string {
	var sb strings.Builder
	sb.WriteString(t.decl.Describe(indent + 4))
	return sb.String()
}

func (t ASTStructDeclarator) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTStructDeclaratorList []ASTStructDeclarator

func (t ASTStructDeclaratorList) Describe(indent int) string {
	var sb strings.Builder
	for i, node := range t {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(node.Describe(indent))
	}
	return sb.String()
}

func (t ASTStructDeclaratorList) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTStructDeclarationList []ASTStructDeclaratorList

func (t ASTStructDeclarationList) Describe(indent int) string {
	var sb strings.Builder
	for i, node := range t {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(node.Describe(indent))
	}
	sb.WriteString("\n")

	return sb.String()
}

func (t ASTStructDeclarationList) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTStructInitilizerList []*ASTAssignment

func (t ASTStructInitilizerList) Describe(indent int) string {
	var sb strings.Builder

	for i, assignment := range t {
		sb.WriteString(assignment.Describe(indent))
		if i != len(t)-1 {
			sb.WriteString(", ")
		}

	}

	return sb.String()
}

func (t ASTStructInitilizerList) GenerateMIPS(w io.Writer, m *MIPS) {}

type ASTStructElement struct {
	structImp *ASTIdentifier
	ident     string
}

func (t ASTStructElement) Describe(indent int) string {
	return fmt.Sprintf("%s.%s", t.structImp.ident, t.ident)
}

func (t ASTStructElement) GenerateMIPS(w io.Writer, m *MIPS) {
	structVar := *m.VariableScopes[len(m.VariableScopes)-1][t.structImp.ident]
	elementIndent := structVar.structure.elementIdents[t.ident]
	elementOffset := structVar.structure.offsets[elementIndent]

	var globalLabel Label
	if structVar.isGlobal {
		globalLabel = structVar.GlobalLabel()

		// Load the address of the global into $v1
		write(w, "lui $v1, %%hi(%s)", globalLabel)
		write(w, "addiu $v1, $v1, %%lo(%s)", globalLabel)
	} else {
		// Put the address of the local into $v1
		write(w, "addiu $v1, $fp, %d", -structVar.fpOffset+elementOffset)
	}

	switch structVar.structure.types[elementIndent].typ {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
		write(w, "lw $v0, %d($fp)", -structVar.fpOffset+elementOffset)
	case VarTypeChar:
		write(w, "lb $v0, %d($fp)", -structVar.fpOffset+elementOffset)
	case VarTypeFloat:
		write(w, "lwc1 $f0, %d($fp)", -structVar.fpOffset+elementOffset)
	case VarTypeDouble:
		write(w, "lwc1 $f0, %d($fp)", -structVar.fpOffset+elementOffset+4)
		write(w, "lwc1 $f1, %d($fp)", -structVar.fpOffset+elementOffset)
	default:
		panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
	}
	m.SetLastType(structVar.structure.types[elementIndent].typ)
}

func genIndent(indent int) string {
	return strings.Repeat(" ", indent)
}

func writeGlobalString(w io.Writer, label Label, value []byte) {
	var sb strings.Builder
	sb.WriteString("\"")
	//for each rune convert them into hex and add \x before hand then add that to the string
	for _, r := range value {
		sb.WriteString(
			fmt.Sprintf("\\x%02x", r),
		)
	}
	sb.WriteString("\\000\"")
	write(w, "%s_data:", label)
	write(w, ".asciz %s", sb.String())
	write(w, "%s:", label)
	write(w, ".word %s_data", label)
}
