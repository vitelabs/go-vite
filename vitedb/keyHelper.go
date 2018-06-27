package vitedb

// DBKP = database key prefix
var (

	DBKP_ACCOUNTBLOCKMETA = []byte("b")

	DBKP_ACCOUNTBLOCK = []byte("c")

	DBKP_TOKENNAME_INDEX = []byte("f")

	DBKP_TOKENSYMBOL_INDEX = []byte("g")

	DBKP_TOKENID_INDEX = []byte("h")
)


func createKey (args... []byte) []byte {
	key := []byte{}
	dot := []byte(".")
	len := len(args)
	for index, arg := range args {
		key = append(key, arg...)
		if index < len - 1 {
			key = append(key, dot...)
		}
	}

	return key
}