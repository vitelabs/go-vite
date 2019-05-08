package util

import (
	"errors"
	"testing"
)

func TestCalcQuotaUsed(t *testing.T) {
	tests := []struct {
		quotaTotal, quotaAddition, quotaLeft, q, qUsed uint64
		err                                            error
	}{
		{15000, 5000, 10001, 0, 4999, nil},
		{15000, 5000, 9999, 1, 5001, nil},
		{10000, 0, 9999, 1, 1, nil},
		{10000, 0, 5000, 5000, 5000, nil},
		{15000, 5000, 5000, 5000, 10000, nil},
		{15000, 5000, 10001, 10000, 4999, ErrOutOfQuota},
		{15000, 5000, 9999, 10000, 5001, ErrOutOfQuota},
		{10000, 0, 9999, 10000, 1, ErrOutOfQuota},
		{10000, 0, 5000, 10000, 5000, ErrOutOfQuota},
		{15000, 5000, 10001, 0, 4999, errors.New("")},
		{15000, 5000, 9999, 1, 5001, errors.New("")},
		{10000, 0, 9999, 1, 1, errors.New("")},
		{10000, 0, 5000, 5000, 5000, errors.New("")},
		{15000, 5000, 5000, 5000, 10000, errors.New("")},
	}
	for i, test := range tests {
		q, qUsed := CalcQuotaUsed(true, test.quotaTotal, test.quotaAddition, test.quotaLeft, test.err)
		if qUsed != test.qUsed || q != test.q {
			t.Fatalf("%v th calculate quota used failed, expected %v:%v, got %v:%v", i, test.q, test.qUsed, q, qUsed)
		}
	}
}

func TestUseQuota(t *testing.T) {
	tests := []struct {
		quotaInit, cost, quotaLeft uint64
		err                        error
	}{
		{100, 100, 0, nil},
		{100, 101, 0, ErrOutOfQuota},
	}
	for _, test := range tests {
		quotaLeft, err := UseQuota(test.quotaInit, test.cost)
		if quotaLeft != test.quotaLeft || err != test.err {
			t.Fatalf("use quota fail, input: %v, %v, expected [%v, %v], got [%v, %v]", test.quotaInit, test.cost, test.quotaLeft, test.err, quotaLeft, err)
		}
	}
}
