%{
package firsttest

var RootTree []int
%}

%union {
  n int
}

%token TOKEN_NUM TOKEN_NEWLINE

// Since everything has a type now, we can get away with doing
// $$.n = $1.n, etc.
%type <n> line exp TOKEN_NUM input

%start input

%%
input:  {  }  /* empty */
       | input line {  RootTree = append(RootTree, $2) }
;

/* ROOT: line { RootTree = $1 }; */

line:     TOKEN_NEWLINE         { $$ = 0; }
       | exp TOKEN_NEWLINE      { $$ = $1; /* fmt.Println($1.n);*/ }
;

exp:     TOKEN_NUM           { $$ = $1; }
       | exp exp '+'   { $$ = $1 + $2; }
       | exp exp '-'   { $$ = $1 - $2; }
       | exp exp '*'   { $$ = $1 * $2; }
       | exp exp '/'   { $$ = $1 / $2; }
	/* Unary minus    */
       | exp 'n'       { $$ = -$1; }
;
%%
