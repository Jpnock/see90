%{
package c90

import (
	"fmt"
	"os"
)

var AST ASTTranslationUnit

func init() {
	yyDebug = 1
	yyErrorVerbose = true
}

func Parse(yylex yyLexer) int {
	return yyParse(yylex)
}
%}

%union {
  n Node
  str string
  typ *ASTType
  assignmentOperator ASTAssignmentOperator
}

%token IDENTIFIER CONSTANT STRING_LITERAL SIZEOF
%token PTR_OP INC_OP DEC_OP LEFT_OP RIGHT_OP LE_OP GE_OP EQ_OP NE_OP
%token AND_OP OR_OP MUL_ASSIGN DIV_ASSIGN MOD_ASSIGN ADD_ASSIGN
%token SUB_ASSIGN LEFT_ASSIGN RIGHT_ASSIGN AND_ASSIGN
%token XOR_ASSIGN OR_ASSIGN TYPE_NAME

%token TYPEDEF EXTERN STATIC AUTO REGISTER
%token CHAR SHORT INT LONG SIGNED UNSIGNED FLOAT DOUBLE CONST VOLATILE VOID
%token STRUCT UNION ENUM ELLIPSIS

%token CASE DEFAULT IF ELSE SWITCH WHILE DO FOR GOTO CONTINUE BREAK RETURN

%start translation_unit
%%

primary_expression
	: IDENTIFIER { $$.n = &ASTIdentifier{ident: $1.str} }
	| CONSTANT { $$.n = &ASTConstant{value: $1.str}}
	| STRING_LITERAL { $$.n = &ASTStringLiteral{value: $1.str} }
	| '(' expression ')' { $$.n = &ASTBrackets{$2.n} }
	;

postfix_expression
	: primary_expression {$$.n = $1.n}
	| postfix_expression '[' expression ']'
	| postfix_expression '(' ')' { $$.n = &ASTFunctionCall{function: $1.n} }
	| postfix_expression '(' argument_expression_list ')' { 
		$$.n = &ASTFunctionCall{
			function: $1.n,
			arguments: $3.n.(ASTArgumentExpressionList),
		}
	}
	| postfix_expression '.' IDENTIFIER
	| postfix_expression PTR_OP IDENTIFIER {}
	| postfix_expression INC_OP {
		$$.n = &ASTExprSuffixUnary{typ: ASTExprSuffixUnaryTypeIncrement, lvalue: $1.n}
	}
	| postfix_expression DEC_OP {
		$$.n = &ASTExprSuffixUnary{typ: ASTExprSuffixUnaryTypeDecrement, lvalue: $1.n}
	}
	;

argument_expression_list
	: assignment_expression { $$.n = ASTArgumentExpressionList{$1.n.(*ASTAssignment)} }
	| argument_expression_list ',' assignment_expression {
		li := $1.n.(ASTArgumentExpressionList)
		li = append(li, $3.n.(*ASTAssignment))
		$$.n = li
	}
	;

unary_expression
	: postfix_expression {$$.n = $1.n}
	| INC_OP unary_expression { 
		$$.n = &ASTExprPrefixUnary{typ: ASTExprPrefixUnaryTypeIncrement, lvalue: $2.n} 
	}
	| DEC_OP unary_expression {
		$$.n = &ASTExprPrefixUnary{typ: ASTExprPrefixUnaryTypeDecrement, lvalue: $2.n}
	}
	| unary_operator cast_expression
	| SIZEOF unary_expression
	| SIZEOF '(' type_name ')'
	;

unary_operator
	: '&'
	| '*'
	| '+'
	| '-'
	| '~'
	| '!'
	;

cast_expression
	: unary_expression {$$.n = $1.n}
	| '(' type_name ')' cast_expression
	;

multiplicative_expression
	: cast_expression {$$.n = $1.n}
	| multiplicative_expression '*' cast_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeMul} }
	| multiplicative_expression '/' cast_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeDiv} }
	| multiplicative_expression '%' cast_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeMod} }
	;

additive_expression
	: multiplicative_expression {$$.n = $1.n}
	| additive_expression '+' multiplicative_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeAdd } }
	| additive_expression '-' multiplicative_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeSub } }
	;

shift_expression
	: additive_expression {$$.n = $1.n}
	| shift_expression LEFT_OP additive_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeLeftShift} }
	| shift_expression RIGHT_OP additive_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeRightShift} }
	;

relational_expression
	: shift_expression {$$.n = $1.n}
	| relational_expression '<' shift_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeLessThan} }
	| relational_expression '>' shift_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeGreaterThan} }
	| relational_expression LE_OP shift_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeLessOrEqual} }
	| relational_expression GE_OP shift_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeGreaterOrEqual} }
	;

equality_expression
	: relational_expression {$$.n = $1.n}
	| equality_expression EQ_OP relational_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeEquality} }
	| equality_expression NE_OP relational_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeNotEquality} }
	;

and_expression
	: equality_expression {$$.n = $1.n}
	| and_expression '&' equality_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeBitwiseAnd} }
	;

exclusive_or_expression
	: and_expression {$$.n = $1.n}
	| exclusive_or_expression '^' and_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeXor} }
	;

inclusive_or_expression
	: exclusive_or_expression {$$.n = $1.n}
	| inclusive_or_expression '|' exclusive_or_expression  { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeBitwiseOr} }
	;

logical_and_expression
	: inclusive_or_expression {$$.n = $1.n}
	| logical_and_expression AND_OP inclusive_or_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeLogicalAnd} }
	;

logical_or_expression
	: logical_and_expression {$$.n = $1.n}
	| logical_or_expression OR_OP logical_and_expression { $$.n = &ASTExprBinary{lhs: $1.n, rhs: $3.n, typ: ASTExprBinaryTypeLogicalOr} }
	;

conditional_expression
	: logical_or_expression {$$.n = $1.n}
	| logical_or_expression '?' expression ':' conditional_expression {panic(":c found ternary")}
	;

assignment_expression
	: conditional_expression {
		$$.n = &ASTAssignment{value: $1.n, tmpAssign: true} 
	}
	| unary_expression assignment_operator assignment_expression { 
		$$.n = &ASTAssignment{ident: $1.str, operator: $2.assignmentOperator, value: $3.n} 
	}
	;

assignment_operator
	: '=' { $$.assignmentOperator = ASTAssignmentOperatorEquals }
	| MUL_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorMulEquals }
	| DIV_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorDivEquals }
	| MOD_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorModEquals }
	| ADD_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorAddEquals }
	| SUB_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorSubEquals }
	| LEFT_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorLeftEquals }
	| RIGHT_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorRightEquals }
	| AND_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorAndEquals }
	| XOR_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorXorEquals }
	| OR_ASSIGN { $$.assignmentOperator = ASTAssignmentOperatorOrEquals }
	;

expression
	: assignment_expression {$$.n = $1.n}
	| expression ',' assignment_expression
	;

constant_expression
	: conditional_expression
	;

declaration
	: declaration_specifiers ';' { 
		fmt.Fprintf(os.Stderr, "Ignoring declaration specifier without init declaration list\n")
		$$.n = ASTDeclaratorList{}
	}
	| declaration_specifiers init_declarator_list ';' {
		for _, entry := range $2.n.(ASTDeclaratorList) {
			entry.typ = $1.typ
		}
		$$.n = $2.n
	}
	;

declaration_specifiers
	: storage_class_specifier
	| storage_class_specifier declaration_specifiers
	| type_specifier { $$.n = $1.typ }
	| type_specifier declaration_specifiers
	| type_qualifier 
	| type_qualifier declaration_specifiers
	;

init_declarator_list
	: init_declarator { $$.n = ASTDeclaratorList{$1.n.(*ASTDecl)} }
	| init_declarator_list ',' init_declarator {
		li := $1.n.(ASTDeclaratorList)
		li = append(li, $3.n.(*ASTDecl))
		$$.n = li
	  }
	;

init_declarator
	: declarator { $$.n = &ASTDecl{ident: $1.str} }
	| declarator '=' initializer { $$.n = &ASTDecl{ident: $1.str, initVal: $3.n} }
	;

storage_class_specifier
	: TYPEDEF
	| EXTERN
	| STATIC
	| AUTO
	| REGISTER
	;

type_specifier
	: VOID { $$.typ = &ASTType{typ: "void"} }
	| CHAR { $$.typ = &ASTType{typ: "char"} }
	| SHORT { 
		// https://stackoverflow.com/a/697531
		$$.typ = &ASTType{typ: "short"}
	  }
	| INT { $$.typ = &ASTType{typ: "int"} }
	| LONG { $$.typ = &ASTType{typ: "long"} }
	| FLOAT { $$.typ = &ASTType{typ: "float"} }
	| DOUBLE { $$.typ = &ASTType{typ: "double"} }
	| SIGNED { $$.typ = &ASTType{typ: "signed"} }
	| UNSIGNED { $$.typ = &ASTType{typ: "unsigned"} }
	| struct_or_union_specifier
	| enum_specifier
	| TYPE_NAME { $$.typ = &ASTType{typ: $1.str} }
	;

struct_or_union_specifier
	: struct_or_union IDENTIFIER '{' struct_declaration_list '}'
	| struct_or_union '{' struct_declaration_list '}'
	| struct_or_union IDENTIFIER
	;

struct_or_union
	: STRUCT
	| UNION
	;

struct_declaration_list
	: struct_declaration
	| struct_declaration_list struct_declaration
	;

struct_declaration
	: specifier_qualifier_list struct_declarator_list ';'
	;

specifier_qualifier_list
	: type_specifier specifier_qualifier_list
	| type_specifier
	| type_qualifier specifier_qualifier_list
	| type_qualifier
	;

struct_declarator_list
	: struct_declarator
	| struct_declarator_list ',' struct_declarator
	;

struct_declarator
	: declarator
	| ':' constant_expression
	| declarator ':' constant_expression
	;

enum_specifier
	: ENUM '{' enumerator_list '}'
	| ENUM IDENTIFIER '{' enumerator_list '}'
	| ENUM IDENTIFIER
	;

enumerator_list
	: enumerator
	| enumerator_list ',' enumerator
	;

enumerator
	: IDENTIFIER
	| IDENTIFIER '=' constant_expression
	;

type_qualifier
	: CONST
	| VOLATILE
	;

declarator
	: pointer direct_declarator
	| direct_declarator { $$.n = $1.n }
	;

direct_declarator
	: IDENTIFIER	{ 
		$$.n = &ASTDirectDeclarator{
			identifier: &ASTIdentifier{
				ident: $1.str,
			},
		}
	}
	| '(' declarator ')'
	| direct_declarator '[' constant_expression ']'
	| direct_declarator '[' ']'
	| direct_declarator '(' parameter_type_list ')' {
		// Function declaration with arguments
		$$.n = &ASTDirectDeclarator{
			decl: $1.n.(*ASTDirectDeclarator),
			parameters: $3.n.(*ASTParameterList),
		}
	}
	| direct_declarator '(' identifier_list ')' {
		// Function declaration for old K&R style funcs
	}
	| direct_declarator '(' ')' {
		// Function declaration with no arguments
		$$.n = &ASTDirectDeclarator{
			decl: $1.n.(*ASTDirectDeclarator),
			parameters: &ASTParameterList{},
		}
	}
	;

pointer
	: '*'
	| '*' type_qualifier_list
	| '*' pointer
	| '*' type_qualifier_list pointer
	;

type_qualifier_list
	: type_qualifier
	| type_qualifier_list type_qualifier
	;

parameter_type_list
	: parameter_list { 
		$$.n = $1.n 
	}
	| parameter_list ',' ELLIPSIS {
		paramList := $1.n.(*ASTParameterList)
		paramList.elipsis = true
		$$.n = paramList
	}
	;

parameter_list
	: parameter_declaration {
		$$.n = &ASTParameterList{
			li: []*ASTParameterDeclaration{
				$1.n.(*ASTParameterDeclaration),
			},
		}
	}
	| parameter_list ',' parameter_declaration {
		li := $1.n.(*ASTParameterList)
		li.li = append(li.li, $3.n.(*ASTParameterDeclaration))
		$$.n = li
	}
	;

parameter_declaration
	: declaration_specifiers declarator {
		$$.n = &ASTParameterDeclaration{
			specifier: $1.n,
			declarator: $2.n,
		}
	}
	| declaration_specifiers abstract_declarator
	| declaration_specifiers {
		$$.n = &ASTParameterDeclaration{
			specifier: $1.n,
		}
	}
	;

identifier_list
	: IDENTIFIER
	| identifier_list ',' IDENTIFIER
	;

type_name
	: specifier_qualifier_list
	| specifier_qualifier_list abstract_declarator
	;

abstract_declarator
	: pointer
	| direct_abstract_declarator
	| pointer direct_abstract_declarator
	;

direct_abstract_declarator
	: '(' abstract_declarator ')'
	| '[' ']'
	| '[' constant_expression ']'
	| direct_abstract_declarator '[' ']'
	| direct_abstract_declarator '[' constant_expression ']'
	| '(' ')'
	| '(' parameter_type_list ')'
	| direct_abstract_declarator '(' ')'
	| direct_abstract_declarator '(' parameter_type_list ')'
	;

initializer
	: assignment_expression {$$.n = $1.n}
	| '{' initializer_list '}'
	| '{' initializer_list ',' '}'
	;

initializer_list
	: initializer
	| initializer_list ',' initializer
	;

statement
	: labeled_statement
	| compound_statement
	| expression_statement
	| selection_statement
	| iteration_statement
	| jump_statement { $$.n = $1.n }
	;

labeled_statement
	: IDENTIFIER ':' statement
	| CASE constant_expression ':' statement
	| DEFAULT ':' statement
	;

compound_statement
	: '{' '}'
	| '{' statement_list '}' { $$.n = $2.n }
	| '{' declaration_list '}' { $$.n = $2.n }
	| '{' declaration_list statement_list '}' { $$.n = &ASTDeclarationStatementLists{decls: $2.n.(ASTDeclaratorList), stmts: $3.n.(ASTStatementList)} }
	;

declaration_list
	: declaration { $$.n = $1.n }
	| declaration_list declaration {
		li := $1.n.(ASTDeclaratorList)
		li = append(li, $2.n.(ASTDeclaratorList)...)
		$$.n = li
	  }
	;

statement_list
	: statement { $$.n = ASTStatementList{$1.n} }
	| statement_list statement {
		li := $1.n.(ASTStatementList)
		li = append(li, $2.n)
		$$.n = li
	  }
	;

expression_statement
	: ';'
	| expression ';'
	;

selection_statement
	: IF '(' expression ')' statement {
		$$.n = &ASTIfStatement{
			condition: $3.n,
			body: $5.n,
			elseBody: nil,
		}
	}
	| IF '(' expression ')' statement ELSE statement {
		$$.n = &ASTIfStatement{
			condition: $3.n,
			body: $5.n,
			elseBody: $7.n,
		}
	}
	| SWITCH '(' expression ')' statement
	;

iteration_statement
	: WHILE '(' expression ')' statement {
		$$.n = &ASTWhileLoop{
			condition: $3.n,
			body: $5.n,
		}
	}
	| DO statement WHILE '(' expression ')' ';' {
		$$.n = &ASTDoWhileLoop{
			condition: $5.n,
			body: $2.n,
		}
	}
	| FOR '(' expression_statement expression_statement ')' statement {
		$$.n = &ASTForLoop{
			initialiser: $3.n,
			condition: $4.n,
			postIterationExpr: nil,
			body: $6.n,
		}
	}
	| FOR '(' expression_statement expression_statement expression ')' statement {
		$$.n = &ASTForLoop{
			initialiser: $3.n,
			condition: $4.n,
			postIterationExpr: $5.n,
			body: $7.n,
		}
	}
	;

jump_statement
	: GOTO IDENTIFIER ';' { 
		$$.n = &ASTGoto{
			label: &ASTIdentifier{ident: $2.str},
		}
	}
	| CONTINUE ';' {
		$$.n = &ASTContinue{}
	}
	| BREAK ';' {
		$$.n = &ASTBreak{}
	}
	| RETURN ';' { $$.n = &ASTReturn{} }
	| RETURN expression ';' { $$.n = &ASTReturn{returnVal: $2.n} }
	;

translation_unit
	: external_declaration { 
		AST = ASTTranslationUnit{
			&ASTNode{inner: $1.n},
		} 
	}
	| translation_unit external_declaration {
		AST = append(AST, &ASTNode{inner: $2.n})
	}
	;

external_declaration
	: function_definition { $$.n = $1.n }
	| declaration
	;

function_definition
	: declaration_specifiers declarator declaration_list compound_statement { panic("Old K&R style function parsed (1)") }// Old K&R style C parameter declarations
	| declaration_specifiers declarator compound_statement { $$.n = &ASTFunction{typ: $1.typ, decl: $2.n.(*ASTDirectDeclarator), body: $3.n} }
	| declarator declaration_list compound_statement { panic("Old K&R style function parsed (2)") }
	| declarator compound_statement { $$.n = &ASTFunction{typ: &ASTType{typ: "int"}, decl: $1.n.(*ASTDirectDeclarator), body: $2.n} } // Function without a type
	;
