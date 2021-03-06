package c90

type VariableScope map[string]*Variable

type VariableScopeStack []VariableScope

func (s *VariableScopeStack) Push(v VariableScope) {
	*s = append(*s, v)
}

func (s *VariableScopeStack) Pop() VariableScope {
	if len(*s) == 0 {
		return nil
	}

	lastElem := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return lastElem
}

func (s *VariableScopeStack) Peek() VariableScope {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}

type LabelScope struct {
	ContinueLabel *Label
	BreakLabel    *Label
}

type LabelScopeStack []LabelScope

func (s *LabelScopeStack) Push(v LabelScope) {
	*s = append(*s, v)
}

func (s *LabelScopeStack) Pop() *LabelScope {
	if len(*s) == 0 {
		return nil
	}

	labelScope := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return &labelScope
}

func (s *LabelScopeStack) Peek() *LabelScope {
	if len(*s) == 0 {
		return nil
	}
	labelScope := (*s)[len(*s)-1]
	return &labelScope
}

type CaseLabel struct {
	switchCase *ASTSwitchCase
	label      Label
}

type CaseLabelScope struct {
	SwitchCase []*CaseLabel
}

type CaseLabelScopeStack []CaseLabelScope

func (s *CaseLabelScopeStack) Push(v CaseLabelScope) {
	*s = append(*s, v)
}

func (s *CaseLabelScopeStack) Pop() *CaseLabelScope {
	if len(*s) == 0 {
		return nil
	}

	labelScope := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return &labelScope
}

func (s *CaseLabelScopeStack) Peek() *CaseLabelScope {
	if len(*s) == 0 {
		return nil
	}
	labelScope := (*s)[len(*s)-1]
	return &labelScope
}

type ReturnScopeStack []Label

func (s *ReturnScopeStack) Push(v Label) {
	*s = append(*s, v)
}

func (s *ReturnScopeStack) Pop() *Label {
	if len(*s) == 0 {
		return nil
	}

	labelScope := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return &labelScope
}

func (s *ReturnScopeStack) Peek() *Label {
	if len(*s) == 0 {
		return nil
	}
	labelScope := (*s)[len(*s)-1]
	return &labelScope
}

type Struct struct {
	astStruct         *ASTStruct
	FlatStructEntries []*StructEntry
	ident             string
	structSize        int
	totalOffsetSize   int
	offsets           map[int]int
	types             map[int]ASTType
	elementIdents     map[string]int
}

type StructScope map[string]*Struct

type StructScopeStack []StructScope

func (s *StructScopeStack) Push(v StructScope) {
	*s = append(*s, v)
}

func (s *StructScopeStack) Pop() StructScope {
	if len(*s) == 0 {
		return nil
	}

	lastElem := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return lastElem
}

func (s *StructScopeStack) Peek() StructScope {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}

type TypeDefScope map[string]*ASTType

type TypeDefScopeStack []TypeDefScope

func (s *TypeDefScopeStack) Push(v TypeDefScope) {
	*s = append(*s, v)
}

func (s *TypeDefScopeStack) Pop() TypeDefScope {
	if len(*s) == 0 {
		return nil
	}

	lastElem := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return lastElem
}

func (s *TypeDefScopeStack) Peek() TypeDefScope {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}
