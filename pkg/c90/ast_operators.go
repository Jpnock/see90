package c90

import (
	"fmt"
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

type ASTExprPrefixUnaryType string

const (
	ASTExprPrefixUnaryTypeIncrement ASTExprPrefixUnaryType = "++"
	ASTExprPrefixUnaryTypeDecrement ASTExprPrefixUnaryType = "--"
	ASTExprPrefixUnaryTypePointer   ASTExprPrefixUnaryType = "->"
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