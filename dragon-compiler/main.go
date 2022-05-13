package main

import (
	"lexer"
	"simple_parser"
	//"inter"
	//"fmt"
)

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
