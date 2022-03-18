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

	sizeStr := sizeConstExpr.Describe(0)
	expr, err := govaluate.NewEvaluableExpression(sizeStr)
	if err != nil {
		panic(err)
	}

	res, err := expr.Evaluate(nil)
	if err != nil {
		panic(err)
	}

	size := int(res.(float64))
	if size < 0 {
		panic("negative array size")
	}

	return &ASTArray{size: size, sizeConstExpr: sizeConstExpr}
}
