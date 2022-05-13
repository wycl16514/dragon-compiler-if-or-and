package inter

import (
	"errors"
	"strconv"
)

type Else struct {
	stmt  *Stmt
	expr  *Expr
	stmt1 *Stmt
	stmt2 *Stmt
}

func NewElse(line uint32, expr *Expr, stmt1 *Stmt, stmt2 *Stmt) *Else {
	if expr.Type().Lexeme != "bool" {
		err := errors.New("bool type required in if")
		panic(err)
	}
	return &Else{
		stmt:  NewStmt(line),
		expr:  expr,
		stmt1: stmt1,
		stmt2: stmt2,
	}
}

func (e *Else) Errors(str string) error {
	return e.stmt.Errors(str)
}

func (e *Else) NewLabel() uint32 {
	return e.stmt.NewLabel()
}

func (e *Else) EmitLabel(i uint32) {
	e.stmt.EmitLabel(i)
}

func (e *Else) Emit(code string) {
	e.stmt.Emit(code)
}

func (e *Else) Gen(_ uint32, end uint32) {
	label1 := e.NewLabel()
	label2 := e.NewLabel()

	e.expr.Jumping(0, label2)
	e.EmitLabel(label1)
	e.stmt1.Gen(label1, end)
	e.Emit("goto L" + strconv.Itoa(int(end)))
	e.EmitLabel(label2)
	e.stmt2.Gen(label2, end)
}
