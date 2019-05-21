package vm

import "github.com/vitelabs/go-vite/vm/util"

func makeStackFunc(pop, push int) stackValidationFunc {
	return func(stack *stack) error {
		if err := stack.require(pop); err != nil {
			return err
		}

		if stack.len()+push-pop > int(stackLimit) {
			return util.ErrStackLimitReached
		}
		return nil
	}
}

func makeDupStackFunc(n int) stackValidationFunc {
	return makeStackFunc(n, n+1)
}

func makeSwapStackFunc(n int) stackValidationFunc {
	return makeStackFunc(n, n)
}
