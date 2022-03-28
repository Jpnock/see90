package c90

import "github.com/Knetic/govaluate"

type ASTArray struct {
	sizeConstExpr Node
	size          int
}

func NewASTArray(sizeConstExpr Node) *ASTArray {
	if sizeConstExpr == nil {
		return &ASTArray{size: 0}
	}

	res := EvaluateConstExpr(sizeConstExpr)
	size := int(res)
	if size < 0 {
		panic("negative array size")
	}

	return &ASTArray{size: size, sizeConstExpr: sizeConstExpr}
}

func EvaluateConstExpr(constExpr Node) float64 {
	expr, err := govaluate.NewEvaluableExpression(constExpr.Describe(0))
	if err != nil {
		panic(err)
	}

	res, err := expr.Evaluate(nil)
	if err != nil {
		panic(err)
	}

	return res.(float64)
}
