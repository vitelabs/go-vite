package utils

func NotNil(err error) {
	if err != nil {
		panic(err)
	}
}
