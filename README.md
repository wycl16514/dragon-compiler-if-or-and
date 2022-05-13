上一节我们完成了if条件判断语句的中间代码生成，我们看到针对if语句的生成代码，我们针对if 条件满足时所要执行的代码赋予了一个跳转标签，同时对if(){...} 右边大括号后面的代码也赋予一个跳转标签，这样我们就能根据if条件判断成立与否进行跳转。


但是上一节实现的if条件判断比较简单，在if()括号里面我们只支持一个算术表达式，事实上它可以通过"||"和"&&"运算符支持更加复杂的表达式，也就是用这些运算符将多个表达式连接在一起，我想每一个写过几行代码的同学都会在if条件判断中使用"||"或者"&&"实现过多个判断条件的组合判断，本节我们看看这种复杂组合判断条件如何实现代码生成。

要实现复杂的组合判断条件，它涉及到比较复杂的语法规则，我们看看对应的语法表达式：
```
        if "(" bool ")" 
		bool -> bool "||"" join | join
		join -> join "&&" equality | equality
		equality -> equality "==" rel | equality != rel | rel 
		rel -> expr < expr | expr <= expr | expr >= expr | expr > expr | expr
```
我们通过具体例子来理解上面规则，形如"a > b", "a <= 3"，”“等这类语句显然对应表达式rel，由于rel包含了expr，因此算术表达式例如"a+2", "c+d"也是与rel的范畴。我们继续往上一层走也就是equality，它是rel根据符号"=="和"!="进行的组合，于是"a!=b", "c+d!=e-f"就属于equality, 注意到 "a>b != c -d"也属于equality对应的规则，虽然这个表达式看起来比较诡异。再往上走equality 对应的表达式可以使用符号"&&"连接起来，于是类似"a>b && c > d"就属于jion的范畴。继续往上走，类似"a>b && c>d || e<f && g > h"则属于boll的范畴。

不知道你能否看出一个规律，那就是越往下对应符号的优先级就越高，例如"a>b && c>d || e < f"，这样的语句在自行时，我们先处理由"&&"连接的表达式，然后再处理"||"连接的表达式，于是给定语句中，编译器要先处理 a>b && c > d的结果，然后再用这个表达式的结果进行"||”运算，这种方法也是编译器确定运算符优先级时常用的方法。

下面我们看看相应代码的实现，上一节我们已经实现了bool函数，在该函数中我们实际上实现的是rel，因为我们在里面直接判断了算术表达式是不是由<, >=, 等这类符号连接的，因此我们把上一节在bool里面的代码抽离出来形成rel函数:
```
func(s *SimpleParser) rel() *inter.Rel {
    expr1 := s.expr()
	var tok *lexer.Token

	switch s.cur_tok.Tag {
	case lexer.LE:
	case lexer.LESS_OPERATOR:
		fallthrough
	case lexer.GE:
		fallthrough
	case lexer.GREATER_OPERATOR:
		fallthrough
	case lexer.NE:
		tok = lexer.NewTokenWithString(s.cur_tok.Tag, s.lexer.Lexeme)
	default:
		tok = nil
	}

	if tok == nil {
		panic("wrong operator in if")
	}
	s.move_forward()
	expr2 := s.expr()
	return inter.NewRel(s.lexer.Line, tok, expr1, expr2)
}
```
根据前面给定的语法规则 bool -> bool "||" join | join ,我们在bool函数中首先执行join函数，如果接下来遇到符号"||"那么就持续再次调用join函数进行解析，于是bool函数代码变成如下模式：
```
func (s *SimpleParser) bool() inter.ExprInterface {
    expr := join() 
	for s.cur_tok.Tag == lexer.OR {
		tok := lexer.NewTokenWithString(s.cur_tok.Tag, s.lexer.Lexeme)
		s.move_forward()
		x := new (tok, expr, s.join())
	}	

	return x
}
```
上面代码中，我们根据语法解析式的逻辑，首先解析join部分，然后判断是否跟着"||"符号，如果是则持续进行join解析，最后生成一个OR节点进行返回，于是这里有两部分我们需要进一步处理，一部分是join函数的实现，一部分是or节点的设计，我们先看后者，在inter目录中创建or.go文件，其实现代码如下：
```
package inter

import (
	"lexer"
)

type Or struct {
	logic *Logic  //它负责处理||， &&， !，等操作符一些共同的逻辑
	expr1 *Expr // "||"前面的表达式
	expr2 *Expr  // "||"后面的表达式
}

func NewOr(line uint32, token *lexer.Token,
	expr1 *Expr, expr2 *Expr) *Or {
	return &Or{
		logic: NewLogic(line, token, expr1, expr2, logicCheckType),
		expr1: expr1,
		expr2: expr2,
	}
}

func (o *Or) Errors(s string) error {
	return o.logic.Errors(s)
}

func (o *Or) NewLabel() uint32 {
	return o.logic.NewLabel()
}

func (o *Or) EmitLabel(l uint32) {
	o.logic.EmitLabel(l)
}

func (o *Or) Emit(code string) {
	o.logic.Emit(code)
}

func (o *Or) Gen() ExprInterface {
	return o.logic.Gen()
}

func (o *Or) Reduce() ExprInterface {
	return o
}

func (o *Or) Type() *Type {
	return o.logic.Type()
}

func (o *Or) ToString() string {
	return o.logic.ToString()
}

func (o *Or) Jumping(t uint32, f uint32) {
	var label uint32
	if t != 0 {
		label = t
	} else {
		label = o.NewLabel()
	}

	o.expr1.Jumping(label, 0)
	o.expr2.Jumping(t, f)
	if t == 0 {
		o.EmitLabel(label)
	}
}

func (o *Or) EmitJumps(test string, t uint32, l uint32) {
	o.logic.EmitJumps(test, t, l)
}

```
在OR节点的实现中，它在创建时需要三个字段，分别是token，它对应运算符"||"，其次是两个算术表达式，分别对应"||"左右两边的算术表达式。在代码实现中需要使用一个名为Logic的对象，它的责任是用于处理"||", "&&", "!"等符号对应表达式需要的一些共同操作，它的实现我们一会再看，现在需要看看它的Jumping代码实现逻辑。

假设我们给定的表达式为"a || b"，那么expr1对应符号a，expr2对应符号b，假设执行Jumping接口调用时输入参数为1,2，那么o.expr1.Jumping(label, 0) 就会生成中间代码:
```
if a goto 1
```
同时o.expr2.Jumping(t,f)生成的代码就是：
```
if b goto 1
goto 2
```
如果两部分是比较复杂的表达式，例如
我们看到在运行"a||b"这个表达式的跳转逻辑时，编译器首先判断第一个表达式是否为真，如果为真则直接跳转到if大括号里面的代码，这里对应的就是标号1，如果为假，那么才继续判断第二个表达式。接下来我们看看Logic节点的实现内容，创建logic.go，实现代码如下：
```
package inter

import (
	"errors"
	"lexer"
	"strconv"
)



/*
实现or, and , !等操作
*/

type Logic struct {
	expr      ExprInterface
	token     *lexer.Token
	expr1     ExprInterface
	expr2     ExprInterface
	expr_type *Type
	line      uint32
}

type CheckType func(type1 *Type, type2 *Type) *Type

func logicCheckType(type1 *Type, type2 *Type) *Type {

	if type1.Lexeme == "bool" && type2.Lexeme == "bool" {
		return type1
	}

	return nil
}

func NewLogic(line uint32, token *lexer.Token,
	expr1 ExprInterface, expr2 ExprInterface, checkType CheckType) *Logic {
	expr_type := checkType(expr1.Type(), expr2.Type())
	if expr_type == nil {
		err := errors.New("type error")
		panic(err)
	}

	return &Logic{
		expr:      NewExpr(line, token, expr_type),
		token:     token,
		expr1:     expr1,
		expr2:     expr2,
		expr_type: expr_type,
		line:      line,
	}
}

func (l *Logic) Errors(s string) error {
	return l.expr.Errors(s)
}

func (l *Logic) NewLabel() uint32 {
	return l.expr.NewLabel()
}

func (l *Logic) EmitLabel(label uint32) {
	l.expr.EmitLabel(label)
}

func (l *Logic) Emit(code string) {
	l.expr.Emit(code)
}

func (l *Logic) Type() *Type {
	return l.expr_type
}

func (l *Logic) Gen() ExprInterface {
	f := l.NewLabel()
	a := l.NewLabel()
	temp := NewTemp(l.line, l.expr_type)
	l.Jumping(0, f)
	l.Emit(temp.ToString() + " = true")
	l.Emit("goto L" + strconv.Itoa(int(a)))
	l.EmitLabel(f)
	l.Emit(temp.ToString() + "=false")
	l.EmitLabel(a)
	return temp
}

func (l *Logic) Reduce() ExprInterface {
	return l
}

func (l *Logic) ToString() string {
	return l.expr1.ToString() + " " + l.token.ToString() + " " + l.expr2.ToString()
}

func (l *Logic) Jumping(t uint32, f uint32) {
	l.expr.Jumping(t, f)
}

func (l *Logic) EmitJumps(test string, t uint32, f uint32) {
	l.expr.EmitJumps(test, t, f)
}

```

它的作用是首先确定符号"||", "&&"， 作用两边的表达式是否为bool类型，只有各个类型才能进行相应操作，也就是目前我们的编译器支持这样的语句"if(a > b || c < d)"，但是暂时不支持"if ( || b)"，事实上对于全功能编译器而言，它其实会在暗地里将a, b等算术表达式转换为bool类型，为了简单起见，我们暂时忽略这种转换。

上面代码中Gen函数的实现逻辑有点诡异，if条件判断语句除了生成跳转代码外，它还能生成其他代码，后面我们在调试代码时会看到它的作用，在这里我们先放一放对它的理解。现在我们看看join函数的实现，在语法表达式里，join对应"&&"操作符的处理，为了简单起见，我们这里直接让它调用rel函数，然后先把当前实现的代码运行起来看看，于是join的实现就是：
```
func (s *SimpleParser) join() inter.ExprInterface {
    return s.rel()
}
```
完成上面代码后，我们在main.go设计一段代码，然后进行编译和代码生成：
```
unc main() {

	source := `{int a; int b; int c; int d;
		        int e;
		        a = 1;
				b = 2;
				c = 3;
				d = 4;
				if (b > a || c < d) {
					e = 2;
				}
				e = 3;
	}`
	my_lexer := lexer.NewLexer(source)
	parser := simple_parser.NewSimpleParser(my_lexer)
	parser.Parse()
}
```
执行后生成的中间代码如下：

![请添加图片描述](https://img-blog.csdnimg.cn/ea43d3b68c844fba9db4de9a81386849.png)
在生成的代码中，需要我们注意的是if语句生成的代码，首先是if b > a goto L9，这里L9标签没有任何代码，因此进入L9后就会直接进入L8,而L8对应的是给变量e赋值2，这与我们代码的逻辑一致。如果执行if b > a后没有跳转到L9,那说明b>a不成立，于是判断第二个条件c < d，这里编译器使用iffalse进行判断，如果c < d不成立，那么直接跳转到L7,而L7对应的是给变量e赋值3,这与我们代码的逻辑一致。

接下来我们看看&&操作符如何实现，首先跟前面的“||”操作符一样，我们需要建立一个名为AND的节点，创建and.go，实现代码如下:
```
package inter

import (
	"lexer"
)

type And struct {
	logic *Logic
	expr1 ExprInterface	
	expr2 ExprInterface
}

func NewAnd(line uint32, token *lexer.Token,
	expr1 ExprInterface, expr2 ExprInterface) *And {
	return &And{
		logic: NewLogic(line, token, expr1, expr2, logicCheckType),
		expr1: expr1,
		expr2: expr2,
	}
}

func (a *And) Errors(s string) error {
	return a.logic.Errors(s)
}

func (a *And) NewLabel() uint32 {
	return a.logic.NewLabel()
}

func (a *And) EmitLabel(l uint32) {
	a.logic.EmitLabel(l)
}

func (a *And) Emit(code string) {
	a.logic.Emit(code)
}

func (a *And) Gen() ExprInterface {
	return a.logic.Gen()
}

func (a *And) Reduce() ExprInterface {
	return a
}

func (a *And) Type() *Type {
	return a.logic.Type()
}

func (a *And) ToString() string {
	return a.logic.ToString()
}

func (a *And) Jumping(t uint32, f uint32) {
	var label uint32
	if f != 0 {
		label = f
	} else {
		f = a.NewLabel()
	}
	a.expr1.Jumping(0, label)
	a.expr2.Jumping(t, f)
	if f == 0 {
		a.EmitLabel(label)
	}
}

func (a *And) EmitJumps(test string, t uint32, l uint32) {
	a.logic.EmitJumps(test, t, l)
}

```
它的逻辑与前面or.go差不多，唯一确保在于Jumping函数生成中间代码时有所不同，它的逻辑跟or正好相反。对于or而言，如果第一个判断成立那么直接进行跳转，但对&&而言，它需要检测的是，如果当前判断不成立，那么就进行跳转，这一点在后面我们调试时会有所体现。接下来我们按照前面描述的语法规则修改一下代码解析的步骤，在list-parser.go中修改如下：
```
func (s *SimpleParser) bool() inter.ExprInterface {
	x := s.join()
	for s.cur_tok.Tag == lexer.OR {
		tok := lexer.NewTokenWithString(s.cur_tok.Tag, s.lexer.Lexeme)
		s.move_forward()
		x = inter.NewOr(s.lexer.Line, tok, x, s.join())
	}

	return x
}

func (s *SimpleParser) join() inter.ExprInterface {
	expr := s.equality()
	var x inter.ExprInterface
	for s.cur_tok.Tag == lexer.AND {
		tok := lexer.NewTokenWithString(s.cur_tok.Tag, s.lexer.Lexeme)
		s.move_forward()
		x = inter.NewAnd(s.lexer.Line, tok, expr, s.equality())
	}

	return x
}

func (s *SimpleParser) equality() inter.ExprInterface {
	expr := s.rel()
	var x inter.ExprInterface
	for s.cur_tok.Tag == lexer.EQ || s.cur_tok.Tag == lexer.NE {
		tok := lexer.NewTokenWithString(s.cur_tok.Tag, s.lexer.Lexeme)
		s.move_forward()
		x = inter.NewRel(s.lexer.Line, tok, expr, s.rel())
	}

	return x
}

func (s *SimpleParser) rel() inter.ExprInterface {
	expr1 := s.expr()
	var tok *lexer.Token

	switch s.cur_tok.Tag {
	case lexer.LE:
	case lexer.LESS_OPERATOR:
		fallthrough
	case lexer.GE:
		fallthrough
	case lexer.GREATER_OPERATOR:
		fallthrough
	default:
		tok = nil
	}

	if tok == nil {
		return expr1
	} else {
		s.move_forward()
	}
	expr2 := s.expr()
	return inter.NewRel(s.lexer.Line, tok, expr1, expr2)
}
```
完成上面代码后，我们创建使用&&操作符的源代码用于解析，在main.go中输入代码如下：
```
func main() {

	source := `{int a; int b; int c; int d;
		        int e;
		        a = 1;
				b = 2;
				c = 3;
				d = 4;
				if (b == a && c != d) {
					e = 2;
				}
				e = 3;
	}`
	my_lexer := lexer.NewLexer(source)
	parser := simple_parser.NewSimpleParser(my_lexer)
	parser.Parse()
}
```
完成后运行起来所得结果如下：
![请添加图片描述](https://img-blog.csdnimg.cn/fdf6da26a77d4b669860f042dbeed9c7.png)
可以看到，编译器在对if(a==b && c!=d)进行代码生成时，创建了两个iffalse语句，这符号逻辑，因为只要有一个判断条件失败，那么跳转就不会进入if语句对应的内部代码，而是直接跳转出if对应大括号后面的代码，因此编译器分别判断条件"b == a" 和"c != d"是否成立，只有有一个不成立就不执行e = 2,直接跳转去执行e = 3，因此我们实现的逻辑没有问题。

更多详细的讲解和调试演示请参看B站视频，搜索coding迪斯尼，代码下载链接: https://pan.baidu.com/s/1JSUikD56p7GQu_a2saUELA 提取码: vvt8.[更多干货](http://m.study.163.com/provider/7600199/index.htm?share=2&shareId=7600199)：http://m.study.163.com/provider/7600199/index.htm?share=2&shareId=7600199
