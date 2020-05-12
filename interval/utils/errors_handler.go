package utils

func Nil(err error) {
	if err != nil {
		panic(err)
	}
}
