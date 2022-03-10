package c90

import (
	"fmt"
	"io"
)

type ASTExprBinaryType string

const (
	ASTExprBinaryTypeMul            ASTExprBinaryType = "*"
	ASTExprBinaryTypeDiv            ASTExprBinaryType = "/"
	ASTExprBinaryTypeMod            ASTExprBinaryType = "%"
	ASTExprBinaryTypeAdd            ASTExprBinaryType = "+"
	ASTExprBinaryTypeSub            ASTExprBinaryType = "-"
	ASTExprBinaryTypeLeftShift      ASTExprBinaryType = "<<"
	ASTExprBinaryTypeRightShift     ASTExprBinaryType = ">>"
	ASTExprBinaryTypeLessThan       ASTExprBinaryType = "<"
	ASTExprBinaryTypeGreaterThan    ASTExprBinaryType = ">"
	ASTExprBinaryTypeLessOrEqual    ASTExprBinaryType = "<="
	ASTExprBinaryTypeGreaterOrEqual ASTExprBinaryType = ">="
	ASTExprBinaryTypeEquality       ASTExprBinaryType = "=="
	ASTExprBinaryTypeNotEquality    ASTExprBinaryType = "!="
	ASTExprBinaryTypeBitwiseAnd     ASTExprBinaryType = "&"
	ASTExprBinaryTypeXor            ASTExprBinaryType = "^"
	ASTExprBinaryTypeBitwiseOr      ASTExprBinaryType = "|"
	ASTExprBinaryTypeLogicalAnd     ASTExprBinaryType = "&&"
	ASTExprBinaryTypeLogicalOr      ASTExprBinaryType = "||"
)

type ASTExprBinary struct {
	lhs Node
	rhs Node
	typ ASTExprBinaryType
}

func (t *ASTExprBinary) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s %s %s", genIndent(indent), t.lhs.Describe(0), t.typ, t.rhs.Describe(0))
}

func write(w io.Writer, format string, args ...interface{}) {
	io.WriteString(
		w,
		fmt.Sprintf(format, args...),
	)
	io.WriteString(w, "\n")
}

func stackPush(w io.Writer, reg string) {
	write(w, "addiu $sp, $sp, -8")
	if reg != "" {
		// TODO: alter sw based on reg type
		write(w, "sw %s, 0($sp)", reg)
	}
}

func stackPop(w io.Writer, reg string) {
	if reg != "" {
		// TODO: alter lw based on reg type
		write(w, "lw %s, 0($sp)", reg)
	}
	write(w, "addiu $sp, $sp, 8")
}

func branchOnCondition(w io.Writer, m *MIPS) {
	trueLabel := m.CreateUniqueLabel("condtion_true")
	finalLabel := m.CreateUniqueLabel("logical_final")
	write(w, "bc1t %s", trueLabel)

	write(w, "addiu $v0, $zero, 0")
	write(w, "j %s", finalLabel)

	write(w, "%s:", trueLabel)
	write(w, "addiu $v0, $zero, 1")

	write(w, "%s:", finalLabel)
}

// TODO: implement for types other than int
func (t *ASTExprBinary) generateLogical(w io.Writer, m *MIPS) {
	// Generate LHS -> result in $v0
	t.lhs.GenerateMIPS(w, m)

	failureLabel := m.CreateUniqueLabel("logical_failure")
	successLabel := m.CreateUniqueLabel("logical_success")
	finalLabel := m.CreateUniqueLabel("logical_final")

	// Do a comparison to check if true/false and short circuit
	switch t.typ {
	case ASTExprBinaryTypeLogicalAnd:
		// Jump to end (failure) if short circuit (false)
		write(w, "beq $zero, $v0, %s", failureLabel)
	case ASTExprBinaryTypeLogicalOr:
		// Jump to end (success) if short circuit (true)
		write(w, "bne $zero, $v0, %s", successLabel)
	default:
		panic("unknown logical function in ASTExprBinary")
	}

	t.rhs.GenerateMIPS(w, m)

	write(w, "%s:", successLabel)
	write(w, "addiu $v0, $zero, 1")
	write(w, "j %s", finalLabel)

	// Jump to this section if the condition is not met
	write(w, "%s:", failureLabel)
	write(w, "addiu $v0, $zero, 0")

	write(w, "%s:", finalLabel)
}

func (t *ASTExprBinary) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: work out actual type
	var varTyp = VarTypeInteger
	switch t.typ {
	case ASTExprBinaryTypeLogicalAnd, ASTExprBinaryTypeLogicalOr:
		// Special case where we need to potentially short circuit, so we cannot
		// always execute RHS.
		t.generateLogical(w, m)
		return
	}

	// Generate LHS -> result in $v0
	t.lhs.GenerateMIPS(w, m)

	switch varTyp {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
		// Store the LHS on the stack
		stackPush(w, "$v0")

		// Generate RHS -> result in $v0
		t.rhs.GenerateMIPS(w, m)

		// TODO: improve this so we don't push/pop to get $v0 into $t1
		stackPush(w, "$v0")

		// Pop the RHS result into $t1
		stackPop(w, "$t1")

		// Pop the LHS result into $t0
		stackPop(w, "$t0")
	case VarTypeFloat, VarTypeDouble:
		stackPush(w, "$f0")

		t.rhs.GenerateMIPS(w, m)

		stackPush(w, "$f0")

		stackPop(w, "$f2")

		stackPop(w, "$f1")
	default:
		panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
	}

	switch t.typ {
	case ASTExprBinaryTypeMul:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong:
			write(w, "mult $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeUnsigned, VarTypeChar:
			write(w, "multu $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeFloat:
			write(w, "mul.s $f0, $f1, $f2")
		case VarTypeDouble:
			write(w, "mul.d $f0, $f1, $f2")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeDiv:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong:
			write(w, "div $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeUnsigned, VarTypeChar:
			write(w, "divu $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeFloat:
			write(w, "div.s $f0, $f1, $f2")
		case VarTypeDouble:
			write(w, "div.d $f0, $f1, $f2")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeMod:
		// TODO: check operation of modulo for negative values
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong:
			write(w, "div $t0, $t1")
			write(w, "mfhi $v0")
		case VarTypeUnsigned, VarTypeChar:
			write(w, "divu $t0, $t1")
			write(w, "mfhi $v0")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeAdd:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "addu $v0, $t0, $t1")
		case VarTypeFloat:
			write(w, "add.s $f0, $f1, $f2")
		case VarTypeDouble:
			write(w, "add.d $f0, $f1, $f2")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeSub:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "subu $v0, $t0, $t1")
		case VarTypeFloat:
			write(w, "sub.s $f0, $f1, $f2")
		case VarTypeDouble:
			write(w, "sub.d $f0, $f1, $f2")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeLeftShift:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeUnsigned, VarTypeChar, VarTypeShort, VarTypeLong:
			write(w, "sllv $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeRightShift:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeUnsigned, VarTypeChar, VarTypeLong:
			write(w, "srlv  $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeLessThan, ASTExprBinaryTypeGreaterOrEqual:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong:
			write(w, "slt $v0, $t0, $t1")
		case VarTypeUnsigned, VarTypeChar:
			write(w, "sltu $v0, $t0, $t1")
		case VarTypeFloat:
			write(w, "c.lt.s $f1, $f2")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.lt.d $f1, $f2")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeGreaterOrEqual {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}

	case ASTExprBinaryTypeGreaterThan, ASTExprBinaryTypeLessOrEqual:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong:
			write(w, "slt $v0, $t1, $t0")
		case VarTypeUnsigned, VarTypeChar:
			write(w, "sltu $v0, $t1, $t0")
		case VarTypeFloat:
			write(w, "c.lt.s $f2, $f1")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.lt.d $f2, $f1")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeLessOrEqual {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}

	case ASTExprBinaryTypeEquality, ASTExprBinaryTypeNotEquality:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			// XOR left with right -> if equal, the result is 0
			write(w, "xor $v0, $t0, $t1")
			// Check (unsigned) whether the integer is less than 1 (i.e. equal to 0)
			write(w, "sltiu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "c.eq.s $f1, $f2")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.eq.d $f1, $f2")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeNotEquality {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}

	case ASTExprBinaryTypeBitwiseAnd:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "AND $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeXor:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "XOR $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeBitwiseOr:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "OR $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	default:
		panic("unsupported ASTExprPrefixUnaryType")
	}
}

type ASTExprPrefixUnaryType string

const (
	ASTExprPrefixUnaryTypeIncrement   ASTExprPrefixUnaryType = "++"
	ASTExprPrefixUnaryTypeDecrement   ASTExprPrefixUnaryType = "--"
	ASTExprPrefixUnaryTypeAddressOf   ASTExprPrefixUnaryType = "&"
	ASTExprPrefixUnaryTypeDereference ASTExprPrefixUnaryType = "*"
	ASTExprPrefixUnaryTypePositive    ASTExprPrefixUnaryType = "+"
	ASTExprPrefixUnaryTypeNegative    ASTExprPrefixUnaryType = "-"
	ASTExprPrefixUnaryTypeNot         ASTExprPrefixUnaryType = "~"
	ASTExprPrefixUnaryTypeInvert      ASTExprPrefixUnaryType = "!"
)

type ASTExprPrefixUnary struct {
	typ    ASTExprPrefixUnaryType
	lvalue Node
}

func (t *ASTExprPrefixUnary) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s%s", genIndent(indent), t.typ, t.lvalue.Describe(0))
}

func (t *ASTExprPrefixUnary) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: maybe change from li.s to cvt.d.s

	// TODO: work out actual type
	var varTyp = VarTypeInteger

	t.lvalue.GenerateMIPS(w, m)

	switch t.typ {
	case ASTExprPrefixUnaryTypeIncrement:
		variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "addiu $v0, $v0, 1")
			write(w, "sw $v0, %d($fp)", -variableOffset)
		case VarTypeFloat:
			write(w, "li.s $f3, 1")
			write(w, "add.s $f0, $f0, $f3")
			write(w, "swc1 $f0, %d($fp)", -variableOffset)
		case VarTypeDouble:
			write(w, "li.d $f3, 1")
			write(w, "add.d $f0, $f0, $f3")
			write(w, "sdc1 $f0, %d($fp)", -variableOffset)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeDecrement:
		variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "addiu $v0, $v0, -1")
			write(w, "sw $v0, %d($fp)", -variableOffset)
		case VarTypeFloat:
			write(w, "li.s $f3, -1")
			write(w, "add.s $f0, $f0, $f3")
			write(w, "swc1 $f0, %d($fp)", -variableOffset)
		case VarTypeDouble:
			write(w, "li.d $f3, -1")
			write(w, "add.d $f0, $f0, $f3")
			write(w, "sdc1 $f0, %d($fp)", -variableOffset)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeInvert:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "sltu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "li.s $f3, 0")
			write(w, "c.eq.s $f0, $f3")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "li.d $f3, 0")
			write(w, "c.eq.d $f0, $f3")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeNegative:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "subu $v0, $zero, $v0")
		case VarTypeFloat:
			write(w, "li.s $f3, 0")
			write(w, "sub.s $f0, $f3, $f0")
		case VarTypeDouble:
			write(w, "li.d $f3, 0")
			write(w, "sub.d $f0, $f3, $f0")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeNot:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "nor $v0, $zero, $v0")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeAddressOf:
		variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset
		write(w, "addiu $v0, $fp, %d", -variableOffset)

	case ASTExprPrefixUnaryTypeDereference:
		// TODO: add info on levels of pointer dereferance you're at
		if _, ok := t.lvalue.(*ASTIdentifier); ok {
			switch varTyp {
			case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
				write(w, "lw $v0, 0($v0)")
			case VarTypeFloat:
				write(w, "l.s $f0, 0($v0)")
			case VarTypeDouble:
				write(w, "l.d $f0, 0($v0)")
			default:
				panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
			}
		} else {
			write(w, "lw $v0, 0($v0)")
		}

	case ASTExprPrefixUnaryTypePositive:
	default:
		panic("unsupported ASTExprPrefixUnaryType")
	}
}

type ASTExprSuffixUnaryType string

const (
	ASTExprSuffixUnaryTypeIncrement ASTExprSuffixUnaryType = "++"
	ASTExprSuffixUnaryTypeDecrement ASTExprSuffixUnaryType = "--"
)

type ASTExprSuffixUnary struct {
	typ    ASTExprSuffixUnaryType
	lvalue Node
}

func (t *ASTExprSuffixUnary) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s%s", genIndent(indent), t.lvalue.Describe(0), t.typ)
}

func (t *ASTExprSuffixUnary) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: work out actual type
	var varTyp = VarTypeInteger

	t.lvalue.GenerateMIPS(w, m)

	// TODO: handle pointers etc
	variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset

	switch t.typ {
	case ASTExprSuffixUnaryTypeIncrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			// The returned value should not be incremented, only the variable.
			write(w, "addiu $v0, $v0, 1")
			write(w, "sw $v0, %d($fp)", -variableOffset)
			write(w, "addiu $v0, $v0, -1")
		case VarTypeFloat:
			write(w, "li.s $f3, 1")
			write(w, "add.s $f0, $f0, $f3")
			write(w, "swc1 $f0, %d($fp)", -variableOffset)
			write(w, "sub.s $f0, $f0, $f3")
		case VarTypeDouble:
			write(w, "li.d $f3, 1")
			write(w, "add.d $f0, $f0, $f3")
			write(w, "sdc1 $f0, %d($fp)", -variableOffset)
			write(w, "sub.d $f0, $f0, $f3")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprSuffixUnaryTypeDecrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			// The returned value should not be decremented, only the variable.
			write(w, "addiu $v0, $v0, -1")
			write(w, "sw $v0, %d($fp)", -variableOffset)
			write(w, "addiu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "li.s $f3, -1")
			write(w, "add.s $f0, $f0, $f3")
			write(w, "swc1 $f0, %d($fp)", -variableOffset)
			write(w, "sub.s $f0, $f0, $f3")
		case VarTypeDouble:
			write(w, "li.d $f3, -1")
			write(w, "add.d $f0, $f0, $f3")
			write(w, "sdc1 $f0, %d($fp)", -variableOffset)
			write(w, "sub.d $f0, $f0, $f3")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	default:
		panic("unsupported ASTExprPrefixUnaryType")
	}
}
