package ledger

import (
	"encoding/json"
	"fmt"

	"github.com/vitelabs/go-vite/common/types"
)

func ExampleHashHeight() {
	var points = make([][2]*HashHeight, 2)

	points[0] = [2]*HashHeight{
		{100, types.Hash{}},
		{200, types.Hash{}},
	}

	points[1] = [2]*HashHeight{
		{201, types.Hash{}},
		{300, types.Hash{}},
	}

	data, err := json.Marshal(points)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", data)
	// Output:
	// [[{"height":100,"hash":"0000000000000000000000000000000000000000000000000000000000000000"},{"height":200,"hash":"0000000000000000000000000000000000000000000000000000000000000000"}],[{"height":201,"hash":"0000000000000000000000000000000000000000000000000000000000000000"},{"height":300,"hash":"0000000000000000000000000000000000000000000000000000000000000000"}]]
}
