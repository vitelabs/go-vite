package net

import (
	"fmt"
	"math/rand"
	"testing"
)

func Test_wait(t *testing.T) {
	const total = 10
	wait := make([]int, 0, total)

	count := rand.Intn(total)

	for i := 0; i < count; i++ {
		wait = append(wait, rand.Intn(total))
	}

	for i := 0; i < 1000; i++ {
		find := rand.Intn(total)

		for i, n := range wait {
			if n == find {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
			}
		}
	}
}

func Test_wait_last(t *testing.T) {
	wait := []int{1, 2, 3, 1, 1, 1}
	total := len(wait)

	for j := 0; j < 4; j++ {
		for i, n := range wait {
			if n == 1 {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
				break
			}
		}

		if len(wait) != (total - j - 1) {
			fmt.Println(len(wait))
			t.Fail()
		}
	}
}

func Test_wait_all(t *testing.T) {
	wait := []int{1, 1, 1, 1, 1, 1}
	total := len(wait)

	for j := 0; j < 4; j++ {
		for i, n := range wait {
			if n == 1 {
				if i != len(wait)-1 {
					copy(wait[i:], wait[i+1:])
				}
				wait = wait[:len(wait)-1]
				break
			}
		}

		if len(wait) != (total - j - 1) {
			fmt.Println(len(wait))
			t.Fail()
		}
	}
}
