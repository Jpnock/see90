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
	switch t.typ {
	case ASTExprBinaryTypeLogicalAnd, ASTExprBinaryTypeLogicalOr:
		// Special case where we need to potentially short circuit, so we cannot
		// always execute RHS.
		t.generateLogical(w, m)
		return
	}

	// Generate LHS -> result in $v0
	t.lhs.GenerateMIPS(w, m)

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

	switch t.typ {
	case ASTExprBinaryTypeMul:
		write(w, "mul $v0, $t0, $t1")
	case ASTExprBinaryTypeDiv:
		write(w, "div $t0, $t1")
	case ASTExprBinaryTypeMod:
		// TODO: check operation of modulo for negative values
		write(w, "div $t0, $t1")
		write(w, "mfhi $v0")
	case ASTExprBinaryTypeAdd:
		write(w, "add $v0, $t0, $t1")
	case ASTExprBinaryTypeSub:
		write(w, "sub $v0, $t0, $t1")
	case ASTExprBinaryTypeLeftShift:
		write(w, "sllv $v0, $t0, $t1")
	case ASTExprBinaryTypeRightShift:
		write(w, "srlv  $v0, $t0, $t1")
	case ASTExprBinaryTypeLessThan:
		write(w, "slt $v0, $t0, $t1")
	case ASTExprBinaryTypeGreaterThan:
		write(w, "slt $v0, $t1, $t0")
	case ASTExprBinaryTypeLessOrEqual:
		// Inverting (left > right) gives (left <= right)
		write(w, "slt $v0, $t1, $t0")
		// Toggle bit (for inversion of condition)
		write(w, "xori $v0, $v0, 1")
	case ASTExprBinaryTypeGreaterOrEqual:
		// Inverting (left < right) gives (left >= right)
		write(w, "slt $v0, $t0, $t1")
		// Toggle bit (for inversion of condition)
		write(w, "xori $v0, $v0, 1")
	case ASTExprBinaryTypeEquality:
		// XOR left with right -> if equal, the result is 0
		write(w, "xor $v0, $t0, $t1")
		// Check (unsigned) whether the integer is less than 1 (i.e. equal to 0)
		write(w, "sltiu $v0, $v0, 1")
	case ASTExprBinaryTypeNotEquality:
		// XOR left with right -> if equal, the result is 0
		write(w, "xor $v0, $t0, $t1")
		// Check (unsigned) whether the integer is less than 1 (i.e. equal to 0)
		write(w, "sltiu $v0, $v0, 1")
		// Invert the condition (greater than 0) => not equal
		write(w, "xori $v0, $v0, 1")
	case ASTExprBinaryTypeBitwiseAnd:
		write(w, "AND $v0, $t0, $t1")
	case ASTExprBinaryTypeXor:
		write(w, "XOR $v0, $t0, $t1")
	case ASTExprBinaryTypeBitwiseOr:
		write(w, "OR $v0, $t0, $t1")
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
	t.lvalue.GenerateMIPS(w, m)

	// TODO: handle pointers etc
	variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset

	switch t.typ {
	case ASTExprPrefixUnaryTypeIncrement:
		write(w, "addiu $v0, $v0, 1")
		write(w, "sw $v0, %d($fp)", -variableOffset)
	case ASTExprPrefixUnaryTypeDecrement:
		write(w, "addiu $v0, $v0, -1")
		write(w, "sw $v0, %d($fp)", -variableOffset)
	case ASTExprPrefixUnaryTypeInvert:
		write(w, "sltu $v0, $v0, 1")
	case ASTExprPrefixUnaryTypeNegative:
		write(w, "subu $v0, $zero, $v0")
	case ASTExprPrefixUnaryTypeNot:
		write(w, "nor $v0, $zero, $v0")
	case ASTExprPrefixUnaryTypeAddressOf:
		write(w, "addiu $v0, $fp, %d", -variableOffset)
	case ASTExprPrefixUnaryTypeDereference:
		write(w, "lw $v0, 0($v0)")
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
	// TODO: fix this so it does assignment before increment
	t.lvalue.GenerateMIPS(w, m)

	// TODO: handle pointers etc
	variableOffset := m.VariableScopes.Peek()[t.lvalue.(*ASTIdentifier).ident].fpOffset

	switch t.typ {
	case ASTExprSuffixUnaryTypeIncrement:
		// The returned value should not be incremented, only the variable.
		write(w, "addiu $v0, $v0, 1")
		write(w, "sw $v0, %d($fp)", -variableOffset)
		write(w, "addiu $v0, $v0, -1")
	case ASTExprSuffixUnaryTypeDecrement:
		// The returned value should not be decremented, only the variable.
		write(w, "addiu $v0, $v0, -1")
		write(w, "sw $v0, %d($fp)", -variableOffset)
		write(w, "addiu $v0, $v0, 1")
	default:
		panic("unsupported ASTExprPrefixUnaryType")
	}
}
