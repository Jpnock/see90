package c90

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// TODO: the following is our own ABI. This needs to be changed to support MIPS
// o32 ABI.
//
// new sp
//
// new frame pointer <- return addr
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]
// 0xfffe0000 <- sp
//
// var2
// var1
// old fp
// 0xffff0000 <- fp (return addr)
// arg 1 [fp + 4]
// arg 2 [fp + 8]
// arg 3 [fp + 0xC]
//
// // GenerateMips -> Function
// // string = body.GenerateMips
// // inspect the context
// // fetch the last offset used
// // subtract last offset + 12 (for ra and old fp, sp) from $sp at start of the function
//
// push 3
// push 2
// push 1
// call function
type ASTFunction struct {
	typ  *ASTType
	decl *ASTDirectDeclarator
	body Node
}

func (t *ASTFunction) Name() string {
	declDescribe := t.decl.Describe(0)
	return declDescribe[:strings.Index(declDescribe, "(")]
}

func (t *ASTFunction) Describe(indent int) string {
	if t == nil {
		panic("ASTFunction is nil")
	}

	indentStr := genIndent(indent)

	declDescribe := t.decl.Describe(0)

	if t.body == nil {
		// TODO: we'd need to generate MIPS for this but just return instantly.
		return fmt.Sprintf("%sfunction (%s) -> %s {}", indentStr, declDescribe, t.typ.Describe(0))
	}
	return fmt.Sprintf("%sfunction (%s) -> %s {\n%s\n}\n", indentStr, declDescribe, t.typ.Describe(0), t.body.Describe(indent+4))
}

func (t *ASTFunction) GenerateMIPS(w io.Writer, m *MIPS) {
	m.NewFunction()
	defer m.EndFunction()

	defer func() {
		// print the lables for strings declared in function
		write(w, ".data")
		for k, v := range m.stringMap {
			writeGlobalString(w, k, v)
		}
		write(w, ".text")
	}()

	// Always return at end of function
	defer write(w, "jr $ra\n")

	returnLabel := m.ReturnScopes.Peek()

	funcName := t.Name()
	write(w, ".text")
	write(w, ".globl %s\n", funcName)
	write(w, "%s:\n", funcName)

	var arguments []*Variable
	nextStackOffset := 0
	for _, param := range t.decl.parameters.li {
		paramType := *param.specifier.(*ASTType)

		// TODO: at the moment, we're assuming all function parameters are
		// identifiers, however this is clearly not the case when you have array
		// parameters.
		directDecl, ok := param.declarator.(*ASTDirectDeclarator)
		if ok {
			v := &Variable{
				fpOffset: -nextStackOffset,
				decl:     nil,
				typ:      paramType,
			}
			m.VariableScopes[len(m.VariableScopes)-1][directDecl.identifier.ident] = v
			arguments = append(arguments, v)
		}

		allocatedSize := 4
		if paramType.typ == VarTypeDouble {
			allocatedSize += 4
		}
		nextStackOffset += allocatedSize
	}

	// TODO: do we need to generate mips
	// t.decl.GenerateMIPS(w, m)

	// Store $sp
	write(w, "move $t7, $sp")

	// Store $fp
	stackPush(w, "$fp", 4)
	defer stackPop(w, "$fp", 4)

	// Move frame pointer to bottom of frame
	// TODO: not ABI compliant
	write(w, "move $fp, $t7")

	bodyBuf := new(bytes.Buffer)
	t.body.GenerateMIPS(bodyBuf, m)

	reserve := m.Context.CurrentStackFramePointerOffset
	write(w, "addiu $sp, $sp, %d", -reserve)
	defer write(w, "addiu $sp, $sp, %d", reserve)

	nextIntReg := 4
	for i, param := range arguments {
		if nextIntReg > 7 || (param.typ.typ == VarTypeDouble && nextIntReg == 7) {
			// We only need to save the first four args max.
			break
		}

		firstParamTyp := arguments[0].typ.typ
		if i < 2 && (firstParamTyp == VarTypeFloat || firstParamTyp == VarTypeDouble) {
			if param.typ.typ == VarTypeFloat {
				if i == 0 {
					write(w, "swc1 $f12, %d($fp)", -param.fpOffset)
				} else {
					write(w, "swc1 $f14, %d($fp)", -param.fpOffset)
				}
				nextIntReg += 1
				continue
			} else if param.typ.typ == VarTypeDouble {
				if i == 0 {
					write(w, "swc1 $f12, %d($fp)", -param.fpOffset+4)
					write(w, "swc1 $f13, %d($fp)", -param.fpOffset)
					nextIntReg += 2
				} else {
					write(w, "swc1 $f14, %d($fp)", -param.fpOffset+4)
					write(w, "swc1 $f15, %d($fp)", -param.fpOffset)
					// As doubles are even register aligned
					nextIntReg += 3
				}
				continue
			}
		}

		if param.typ.typ == VarTypeDouble {
			if nextIntReg%2 != 0 {
				// As doubles are even register aligned
				nextIntReg += 1
			}
			write(w, "sw $%d, %d($fp)", nextIntReg, -param.fpOffset)
			write(w, "sw $%d, %d($fp)", nextIntReg+1, -param.fpOffset)
			nextIntReg += 2
		} else {
			write(w, "sw $%d, %d($fp)", nextIntReg, -param.fpOffset)
			nextIntReg += 1
		}
	}

	write(w, "%s", bodyBuf.String())

	write(w, "%s:", *returnLabel)
}

type ASTFunctionCall struct {
	// primary_expresion node
	function  Node
	arguments ASTArgumentExpressionList
}

func (t *ASTFunctionCall) Describe(indent int) string {
	if t == nil {
		return ""
	}
	var sb strings.Builder
	for i, arg := range t.arguments {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.Describe(0))
	}
	return fmt.Sprintf("%s%s(%s)", genIndent(indent), t.function.Describe(0), sb.String())
}

func (t *ASTFunctionCall) GenerateMIPS(w io.Writer, m *MIPS) {
	// Need to do this as the very start/last thing
	// to prevent it interfering with the call stack.
	stackPush(w, "$ra", 4)
	defer stackPop(w, "$ra", 4)

	const regTypeFP = "fp"
	const regTypeInt = "int"
	firstRegisterType := regTypeInt

	overflowArgsStackPopAmount := 0
	lastIntRegisterUsed := 3
	numBytesUsed := 0

	// TODO: decide when to switch to stack based on 4x4 byte arguments
	for i, arg := range t.arguments {
		arg.GenerateMIPS(w, m)

		if numBytesUsed >= 16 || lastIntRegisterUsed >= 7 {
			// Put variables on stack as we've overflowed the register space
			// available.
			switch m.LastType {
			case VarTypeFloat:
				overflowArgsStackPopAmount += 4
				stackPushFP(w, "$f0")
			case VarTypeDouble:
				overflowArgsStackPopAmount += 8
				stackPushFP(w, "$f0", "$f1")
			default:
				// TODO: Sizing is wrong for some types (and probably argument ordering)
				overflowArgsStackPopAmount += 4
				write(w, "addiu $sp, $sp, -4")
				write(w, "sw $v0, 0($sp)")
			}
			continue
		}

		if i == 0 && (m.LastType == VarTypeFloat || m.LastType == VarTypeDouble) {
			// Check if we need to handle the edgecase
			firstRegisterType = regTypeFP
		}

		nextIntReg := lastIntRegisterUsed + 1

		if firstRegisterType == regTypeFP && i < 2 {
			if m.LastType == VarTypeFloat {
				if nextIntReg == 4 {
					write(w, "mov.s $f12, $f0")
				} else {
					write(w, "mov.s $f14, $f0")
				}

				lastIntRegisterUsed += 1
				numBytesUsed += 4

				// Process next arg
				continue
			} else if m.LastType == VarTypeDouble {
				if nextIntReg == 4 {
					write(w, "mov.s $f12, $f0")
					write(w, "mov.s $f13, $f1")
				} else {
					write(w, "mov.s $f14, $f0")
					write(w, "mov.s $f15, $f1")
				}

				lastIntRegisterUsed += 2
				numBytesUsed += 8

				// Process next arg
				continue
			}

			// We're no longer dealing with an FP arg, so handle this below.
		}

		// Everything from herein goes into int registers
		switch m.LastType {
		case VarTypeInteger, VarTypeSigned, VarTypeShort, VarTypeLong, VarTypeUnsigned, VarTypeChar, VarTypeString:
			write(w, "move $%d, $v0", nextIntReg)
			numBytesUsed += 4
		case VarTypeFloat:
			write(w, "mfc1 $%d, $f0", nextIntReg)
			numBytesUsed += 4
		case VarTypeDouble:
			if nextIntReg%2 != 0 {
				// Needs to be even aligned for some reason.
				nextIntReg += 1
			}
			write(w, "mfc1 $%d, $f0", nextIntReg+1)
			write(w, "mfc1 $%d, $f1", nextIntReg)

			// We use two registers, so increment this here.
			nextIntReg += 1
			numBytesUsed += 8
		default:
			panic("unknown function call arg type")
		}

		lastIntRegisterUsed = nextIntReg
	}

	write(w, "addiu $sp, $sp, -16")
	write(w, "jal %s", t.FunctionName())
	write(w, "addiu $sp, $sp, 16")

	if overflowArgsStackPopAmount > 0 {
		write(w, "addiu $sp, $sp, %d", overflowArgsStackPopAmount)
	}
}

func (t *ASTFunctionCall) FunctionName() string {
	return t.function.Describe(0)
}
