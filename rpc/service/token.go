package service


type Token struct {}

func (*Token) GetList (tokenQuery *TokenQuery, tokenList interface{}) error {
	return nil
}