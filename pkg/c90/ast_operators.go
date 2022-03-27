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

func stackPushFP(w io.Writer, registers ...string) {
	if len(registers) > 2 {
		panic("bad stackPushFP")
	}

	if len(registers) == 2 {
		write(w, "addiu $sp, $sp, -8")
		write(w, "swc1 %s, 4($sp)", registers[0])
		write(w, "swc1 %s, 0($sp)", registers[1])
		return
	}

	write(w, "addiu $sp, $sp, -4")
	write(w, "swc1 %s, 0($sp)", registers[0])
}

func stackPopFP(w io.Writer, registers ...string) {
	if len(registers) > 2 {
		panic("bad stackPopFP")
	}

	if len(registers) == 2 {
		write(w, "lwc1 %s, 4($sp)", registers[0])
		write(w, "lwc1 %s, 0($sp)", registers[1])
		write(w, "addiu $sp, $sp, 8")
		return
	}

	write(w, "lwc1 %s, 0($sp)", registers[0])
	write(w, "addiu $sp, $sp, 4")
}

func stackPush(w io.Writer, reg string, size int) {
	write(w, "addiu $sp, $sp, -8")
	if reg != "" {
		// TODO: alter sw based on reg type
		switch size {
		case 2:
			write(w, "sb %s, 0($sp)", reg)
		case 4:
			write(w, "sw %s, 0($sp)", reg)
		default:
			write(w, "un implemented size push")
		}

	}
}

func stackPop(w io.Writer, reg string, size int) {
	if reg != "" {
		// TODO: alter lw based on reg type
		switch size {
		case 2:
			write(w, "lb %s, 0($sp)", reg)
		case 4:
			write(w, "lw %s, 0($sp)", reg)
		default:
			write(w, "un implemented size pop")
		}
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
	checkFloatOrDoubleCondition(w, m)

	failureLabel := m.CreateUniqueLabel("logical_failure")
	successLabel := m.CreateUniqueLabel("logical_success")
	endLabel := m.CreateUniqueLabel("logical_end")

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

	// Generate RHS -> result in $v0
	t.rhs.GenerateMIPS(w, m)
	checkFloatOrDoubleCondition(w, m)

	switch t.typ {
	case ASTExprBinaryTypeLogicalAnd:
		// Jump to end (failure) if short circuit (false)
		write(w, "beq $zero, $v0, %s", failureLabel)
		// Both LHS and RHS are non-zero, so jump to success
		write(w, "j %s", successLabel)
	case ASTExprBinaryTypeLogicalOr:
		// Jump to end (success) if short circuit (true)
		write(w, "bne $zero, $v0, %s", successLabel)
	default:
		panic("unknown logical function in ASTExprBinary")
	}

	// Jump to this section if the condition is not met
	write(w, "%s:", failureLabel)
	write(w, "addiu $v0, $zero, 0")
	write(w, "j %s", endLabel)

	write(w, "%s:", successLabel)
	write(w, "addiu $v0, $zero, 1")
	write(w, "%s:", endLabel)
	m.LastType = VarTypeInteger
}

func (t *ASTExprBinary) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: work out actual type
	switch t.typ {
	case ASTExprBinaryTypeLogicalAnd, ASTExprBinaryTypeLogicalOr:
		// Special case where we need to potentially short circuit, so we cannot
		// always execute RHS.
		t.generateLogical(w, m)
		return
	}

	// Generate LHS -> result in $v0
	t.lhs.GenerateMIPS(w, m)

	var varTyp = m.LastType

	switch varTyp {
	case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeString:
		// Store the LHS on the stack
		stackPush(w, "$v0", 4)

		// Generate RHS -> result in $v0
		t.rhs.GenerateMIPS(w, m)

		// TODO: improve this so we don't push/pop to get $v0 into $t1
		stackPush(w, "$v0", 4)

		// Pop the RHS result into $t1
		stackPop(w, "$t1", 4)

		// Pop the LHS result into $t0
		stackPop(w, "$t0", 4)
	case VarTypeFloat:
		stackPushFP(w, "$f0")
		t.rhs.GenerateMIPS(w, m)
		stackPushFP(w, "$f0")

		stackPopFP(w, "$f4")
		stackPopFP(w, "$f2")
	case VarTypeDouble:
		stackPushFP(w, "$f0", "$f1")
		t.rhs.GenerateMIPS(w, m)
		stackPushFP(w, "$f0", "$f1")

		stackPopFP(w, "$f4", "$f5")
		stackPopFP(w, "$f2", "$f3")
	case VarTypeChar:
		stackPush(w, "$v0", 2)

		t.rhs.GenerateMIPS(w, m)

		stackPush(w, "$v0", 2)

		stackPop(w, "$t1", 2)

		stackPop(w, "$t0", 2)
	default:
		panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
	}

	switch t.typ {
	case ASTExprBinaryTypeMul:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeChar:
			write(w, "mult $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeUnsigned:
			write(w, "multu $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeFloat:
			write(w, "mul.s $f0, $f2, $f4")
		case VarTypeDouble:
			write(w, "mul.d $f0, $f2, $f4")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeDiv:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeChar:
			write(w, "div $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeUnsigned:
			write(w, "divu $t0, $t1")
			write(w, "mflo $v0")
		case VarTypeFloat:
			write(w, "div.s $f0, $f2, $f4")
		case VarTypeDouble:
			write(w, "div.d $f0, $f2, $f4")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeMod:
		// TODO: check operation of modulo for negative values
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeChar:
			write(w, "div $t0, $t1")
			write(w, "mfhi $v0")
		case VarTypeUnsigned:
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
			write(w, "add.s $f0, $f2, $f4")
		case VarTypeDouble:
			write(w, "add.d $f0, $f2, $f4")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeSub:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "subu $v0, $t0, $t1")
		case VarTypeFloat:
			write(w, "sub.s $f0, $f2, $f4")
		case VarTypeDouble:
			write(w, "sub.d $f0, $f2, $f4")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprBinaryTypeLeftShift:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeUnsigned, VarTypeChar, VarTypeShort, VarTypeLong:
			write(w, "sllv $v0, $t0, $t1")
		case VarTypeFloat, VarTypeDouble:
			panic("not allowed operation on type float or double")
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
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeChar:
			write(w, "slt $v0, $t0, $t1")
		case VarTypeUnsigned:
			write(w, "sltu $v0, $t0, $t1")
		case VarTypeFloat:
			write(w, "c.lt.s $f2, $f4")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.lt.d $f2, $f4")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeGreaterOrEqual {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}
		m.LastType = VarTypeInteger

	case ASTExprBinaryTypeGreaterThan, ASTExprBinaryTypeLessOrEqual:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeChar:
			write(w, "slt $v0, $t1, $t0")
		case VarTypeUnsigned:
			write(w, "sltu $v0, $t1, $t0")
		case VarTypeFloat:
			write(w, "c.lt.s $f4, $f2")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.lt.d $f4, $f2")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeLessOrEqual {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}
		m.LastType = VarTypeInteger

	case ASTExprBinaryTypeEquality, ASTExprBinaryTypeNotEquality:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar, VarTypeString:
			// XOR left with right -> if equal, the result is 0
			write(w, "xor $v0, $t0, $t1")
			// Check (unsigned) whether the integer is less than 1 (i.e. equal to 0)
			write(w, "sltiu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "c.eq.s $f2, $f4")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "c.eq.d $f2, $f4")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		if t.typ == ASTExprBinaryTypeNotEquality {
			// Invert the condition (greater than 0) => not equal
			write(w, "xori $v0, $v0, 1")
		}
		m.LastType = VarTypeInteger

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
	ASTExprPrefixUnaryTypeSizeOf      ASTExprPrefixUnaryType = "sizeof"
)

type ASTExprPrefixUnary struct {
	typ    ASTExprPrefixUnaryType
	lvalue Node
}

func (t *ASTExprPrefixUnary) Describe(indent int) string {
	if t == nil {
		return ""
	}
	if t.typ == ASTExprPrefixUnaryTypeSizeOf {
		return fmt.Sprintf("%s%s(%s)", genIndent(indent), t.typ, t.lvalue.Describe(0))
	}
	return fmt.Sprintf("%s%s%s", genIndent(indent), t.typ, t.lvalue.Describe(0))
}

func (t *ASTExprPrefixUnary) GenerateMIPS(w io.Writer, m *MIPS) {
	// TODO: maybe change from li.s to cvt.d.s

	// TODO: work out actual type
	t.lvalue.GenerateMIPS(w, m)

	var varTyp = m.LastType

	switch t.typ {
	case ASTExprPrefixUnaryTypeIncrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
			write(w, "addiu $v0, $v0, 1")
			write(w, "sw $v0, 0($v1)")
		case VarTypeChar:
			write(w, "addiu $v0, $v0, 1")
			write(w, "sb $v0, 0($v1)")
		case VarTypeFloat:
			write(w, "li.s $f10, 1")
			write(w, "add.s $f0, $f0, $f10")
			write(w, "swc1 $f0, 0($v1)")
		case VarTypeDouble:
			write(w, "li.d $f10, 1")
			write(w, "add.d $f0, $f0, $f10")
			write(w, "swc1 $f0, 4($v1)")
			write(w, "swc1 $f1, 0($v1)")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeDecrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
			write(w, "addiu $v0, $v0, -1")
			write(w, "sw $v0, 0($v1)")
		case VarTypeChar:
			write(w, "addiu $v0, $v0, -1")
			write(w, "sb $v0, 0($v1)")
		case VarTypeFloat:
			write(w, "li.s $f10, -1")
			write(w, "add.s $f0, $f0, $f10")
			write(w, "swc1 $f0, 0($v1)")
		case VarTypeDouble:
			write(w, "li.d $f10, -1")
			write(w, "add.d $f0, $f0, $f10")
			write(w, "swc1 $f0, 4($v1)")
			write(w, "swc1 $f1, 0($v1)")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprPrefixUnaryTypeInvert:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "sltu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "li.s $f10, 0")
			write(w, "c.eq.s $f0, $f10")
			branchOnCondition(w, m)
		case VarTypeDouble:
			write(w, "li.d $f10, 0")
			write(w, "c.eq.d $f0, $f10")
			branchOnCondition(w, m)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		m.LastType = VarTypeInteger

	case ASTExprPrefixUnaryTypeNegative:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar:
			write(w, "subu $v0, $zero, $v0")
		case VarTypeFloat:
			write(w, "li.s $f10, 0")
			write(w, "sub.s $f0, $f10, $f0")
		case VarTypeDouble:
			write(w, "li.d $f10, 0")
			write(w, "sub.d $f0, $f10, $f0")
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
		write(w, "addu $v0, $zero, $v1")

	case ASTExprPrefixUnaryTypeDereference:
		write(w, "addu $v1, $v0, $zero")
		// TODO: add info on levels of pointer dereferance you're at
		if _, ok := t.lvalue.(*ASTIdentifier); ok {
			switch varTyp {
			case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
				write(w, "lw $v0, 0($v0)")
			case VarTypeString, VarTypeChar:
				write(w, "lb $v0, 0($v0)")
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
	case ASTExprPrefixUnaryTypeSizeOf:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeFloat:
			write(w, "li $v0, 4")
		case VarTypeChar:
			write(w, "li $v0, 1")
		case VarTypeDouble:
			write(w, "li $v0, 8")
		case VarTypeStruct:
			structVar := m.VariableScopes[len(m.VariableScopes)-1][t.lvalue.(*ASTBrackets).Node.(ASTExpression)[0].value.(*ASTIdentifier).ident]
			write(w, "li $v0, %d", structVar.structure.structSize)
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}
		m.LastType = VarTypeInteger

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
	// TODO: work out actual type this probs doesnt work

	t.lvalue.GenerateMIPS(w, m)

	var varTyp = m.LastType
	// TODO: handle global variables

	switch t.typ {
	case ASTExprSuffixUnaryTypeIncrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
			// The returned value should not be incremented, only the variable.
			write(w, "addiu $v0, $v0, 1")
			write(w, "sw $v0, 0($v1)")
			write(w, "addiu $v0, $v0, -1")
		case VarTypeChar:
			write(w, "addiu $v0, $v0, 1")
			write(w, "sb $v0, 0($v1)")
			write(w, "addiu $v0, $v0, -1")
		case VarTypeFloat:
			write(w, "li.s $f10, 1")
			write(w, "add.s $f0, $f0, $f10")
			write(w, "swc1 $f0, 0($v1)")
			write(w, "sub.s $f0, $f0, $f10")
		case VarTypeDouble:
			write(w, "li.d $f10, 1")
			write(w, "add.d $f0, $f0, $f10")
			write(w, "swc1 $f0, 4($v1)")
			write(w, "swc1 $f1, 0($v1)")
			write(w, "sub.d $f0, $f0, $f10")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	case ASTExprSuffixUnaryTypeDecrement:
		switch varTyp {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned:
			// The returned value should not be decremented, only the variable.
			write(w, "addiu $v0, $v0, -1")
			write(w, "sw $v0, 0($v1)")
			write(w, "addiu $v0, $v0, 1")
		case VarTypeChar:
			write(w, "addiu $v0, $v0, -1")
			write(w, "sb $v0, 0($v1)")
			write(w, "addiu $v0, $v0, 1")
		case VarTypeFloat:
			write(w, "li.s $f10, -1")
			write(w, "add.s $f0, $f0, $f10")
			write(w, "swc1 $f0, 0($v1)")
			write(w, "sub.s $f0, $f0, $f10")
		case VarTypeDouble:
			write(w, "li.d $f10, -1")
			write(w, "add.d $f0, $f0, $f10")
			write(w, "swc1 $f0, 4($v1)")
			write(w, "swc1 $f1, 0($v1)")
			write(w, "sub.d $f0, $f0, $f10")
		default:
			panic("not yet implemented code gen on binary expressions for these types: VarTypeTypeName, VarTypeVoid")
		}

	default:
		panic("unsupported ASTExprPrefixUnaryType")
	}
}

type ASTIndexedExpression struct {
	lvalue Node
	index  Node
}

func (t *ASTIndexedExpression) Describe(indent int) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("%s%s[%s]", genIndent(indent), t.lvalue.Describe(0), t.index.Describe(0))
}

func (t *ASTIndexedExpression) GenerateMIPS(w io.Writer, m *MIPS) {
	// Put index into $v0
	t.index.GenerateMIPS(w, m)
	stackPush(w, "$v0", 4)

	// Put lvalue into $v0
	t.lvalue.GenerateMIPS(w, m)

	// Index now in $t0
	stackPop(w, "$t0", 4)

	// TODO: alter based on type (currently + 4x$t0 for int)
	write(w, "addu $v0, $v0, $t0")
	write(w, "addu $v0, $v0, $t0")
	write(w, "addu $v0, $v0, $t0")
	write(w, "addu $v0, $v0, $t0")

	// TODO: change based on type
	switch m.LastType {
	case VarTypeString:
		write(w, "lb $v0, 0($v0)")
		m.LastType = VarTypeChar
	default:
		write(w, "lw $v0, 0($v0")
	}
}
