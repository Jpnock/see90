package c90

import "fmt"

type Label string

type Variable struct {
	// fpOffset is the amount of subtract from fp to access the variable.
	fpOffset int
	decl     *ASTDecl
}

type MIPSContext struct {
	CurrentStackFramePointerOffset int
}

func (m *MIPSContext) GetNewLocalOffset() int {
	// TODO: change this size depending on the type of variable
	m.CurrentStackFramePointerOffset += 8
	return m.CurrentStackFramePointerOffset
}

type MIPS struct {
	VariableScopes  VariableScopeStack
	Context         *MIPSContext
	LabelScopes     LabelScopeStack
	CaseLabelScopes CaseLabelScopeStack

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

// NewCaseLabelScope adds a new case label scope to the stack. Unlike other
// scopes, it does not copy in the previous values on the stack.
func (m *MIPS) NewSwitchStatement() (bottomLabel Label) {
	bottomLabel = m.CreateUniqueLabel("switch_bottom")
	m.CaseLabelScopes.Push(CaseLabelScope{})
	m.LabelScopes.Push(LabelScope{BreakLabel: &bottomLabel})
	m.NewVariableScope()
	return
}

func (m *MIPS) EndSwitchStatement() {
	m.VariableScopes.Pop()
	m.LabelScopes.Pop()
	m.CaseLabelScopes.Pop()
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

func (m *MIPS) EndFunction() {
	m.VariableScopes.Pop()
}