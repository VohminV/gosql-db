package parser

// Lexer выполняет лексический анализ входной строки.
type Lexer struct {
	input string
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, pos: 0}
}

// NextToken возвращает следующую лексему.
func (l *Lexer) NextToken() Token {
	// Реализация токенизации
	return Token{}
}