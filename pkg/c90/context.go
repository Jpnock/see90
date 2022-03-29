package c90

import (
	"fmt"
)

type Label string

type Variable struct {
	// fpOffset is the amount of subtract from fp to access the variable.
	fpOffset          int
	decl              *ASTDecl
	directDecl        *ASTDirectDeclarator
	typ               ASTType
	label             *Label
	isGlobal          bool
	isLocalDataString bool
	enum              *ASTEnumEntry
	structure         *Struct
}

func (v *Variable) IsPointer() bool {
	if v.directDecl == nil {
		return false
	}
	return v.directDecl.pointerDepth > 0
}

func (v *Variable) IsArray() bool {
	if v.directDecl == nil {
		return false
	}
	return v.directDecl.array != nil
}

func (v *Variable) GlobalLabel() Label {
	if !v.isGlobal {
		panic("variable not global")
	}
	return Label("__global_var__" + v.directDecl.Identifier().ident)
}

type MIPSContext struct {
	CurrentStackFramePointerOffset int
}

func (m *MIPSContext) GetNewLocalOffset() int {
	// TODO: change this size depending on the type of variable
	m.CurrentStackFramePointerOffset += 8
	return m.CurrentStackFramePointerOffset
}

func (m *MIPSContext) GetNewLocalOffsetWithMinSize(reserve int) int {
	// Align to 8 bytes for the ABI
	mod := reserve % 8
	if mod != 0 {
		reserve += 8 - mod
	}
	m.CurrentStackFramePointerOffset += reserve
	return m.CurrentStackFramePointerOffset
}

type MIPS struct {
	VariableScopes    VariableScopeStack
	Context           *MIPSContext
	LabelScopes       LabelScopeStack
	CaseLabelScopes   CaseLabelScopeStack
	ReturnScopes      ReturnScopeStack
	StructScopes      StructScopeStack
	TypeDefScopes     TypeDefScopeStack
	StructElementName string
	TopStruct         string
	stringMap         map[Label][]byte
	lastLabel         Label

	lastType     VarType
	pointerLevel int
	indexLevel   int

	uniqueLabelNumber uint
}

func NewMIPS() *MIPS {
	return &MIPS{
		VariableScopes: VariableScopeStack{
			// Global scope is always the first level
			VariableScope{},
		},
		StructScopes: StructScopeStack{
			StructScope{},
		},
		TypeDefScopes: TypeDefScopeStack{
			// Global scope is always the first level
			TypeDefScope{},
		},
		Context:           &MIPSContext{},
		LabelScopes:       nil,
		stringMap:         make(map[Label][]byte),
		lastType:          VarTypeInvalid,
		uniqueLabelNumber: 0,
	}
}

func (m *MIPS) LastType() VarType {
	if m.pointerLevel != 0 {
		return VarTypeUnsigned
	}
	if m.pointerLevel == 1 && m.lastType == VarTypeChar {
		return VarTypeString
	}
	return m.lastType
}

func (m *MIPS) SetLastType(typ VarType) {
	//	log.Printf("setting type to %v\n", typ)
	//debug.PrintStack()

	if typ == VarTypeString {
		m.lastType = VarTypeChar
		m.pointerLevel = 1
		return
	}

	m.lastType = typ
	m.pointerLevel = 0
}

// CreateUniqueLabel takes the provided name and returns a unique label, using
// this name.
func (m *MIPS) CreateUniqueLabel(name string) Label {
	label := fmt.Sprintf("__label__%s__%d__", name, m.uniqueLabelNumber)
	m.uniqueLabelNumber++
	return Label(label)
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

func (m *MIPS) NewStructScope() {
	// Create a new scope and copy the last scope into it
	newScope := make(StructScope)
	top := m.StructScopes.Peek()
	for k, v := range top {
		newScope[k] = v
	}
	m.StructScopes.Push(newScope)
}

func (m *MIPS) NewTypeDefScope() {
	// Create a new scope and copy the last scope into it
	newScope := make(TypeDefScope)
	top := m.TypeDefScopes.Peek()
	for k, v := range top {
		newScope[k] = v
	}
	m.TypeDefScopes.Push(newScope)
}

// NewLabelScope adds a new label scope to the stack and copies all of the
// previous variables into it.
func (m *MIPS) NewLabelScope(l LabelScope) {
	// TODO: copy the last scope into it when we have other labels
	m.LabelScopes.Push(l)
}

// NewCaseLabelScope adds a new case label scope to the stack. Unlike other
// scopes, it does not copy in the previous values on the stack.
func (m *MIPS) NewSwitchStatement() (bottomLabel Label) {
	bottomLabel = m.CreateUniqueLabel("switch_bottom")
	m.CaseLabelScopes.Push(CaseLabelScope{})
	m.LabelScopes.Push(LabelScope{BreakLabel: &bottomLabel})
	m.NewVariableScope()
	m.NewStructScope()
	m.NewTypeDefScope()
	return
}

func (m *MIPS) EndSwitchStatement() {
	m.VariableScopes.Pop()
	m.LabelScopes.Pop()
	m.StructScopes.Pop()
	m.TypeDefScopes.Pop()
	m.CaseLabelScopes.Pop()
}

// NewFunction resets context variables relating to the current function being
// generated.
func (m *MIPS) NewFunction() {
	// TODO: change this if we change our stack frame
	const fp = 8
	m.Context.CurrentStackFramePointerOffset = fp

	//clear map of strings declared in last function
	m.stringMap = map[Label][]byte{}

	m.NewVariableScope()
	m.NewStructScope()
	m.NewTypeDefScope()
	m.ReturnScopes.Push(m.CreateUniqueLabel("function_return"))
}

func (m *MIPS) EndFunction() {
	m.VariableScopes.Pop()
	m.StructScopes.Pop()
	m.TypeDefScopes.Pop()
	m.ReturnScopes.Pop()
	m.stringMap = map[Label][]byte{}
}

func (m *MIPS) sizeOfType(typ VarType, pointer bool) int {
	if pointer {
		return 4
	}

	switch typ {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeFloat, VarTypeString:
		return 4
	case VarTypeChar:
		return 1
	case VarTypeDouble:
		return 8
	default:
		panic(fmt.Sprintf("unknown sizeof type: %s", typ))
	}
}
