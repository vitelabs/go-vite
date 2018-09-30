package quota

import (
	"errors"
	"testing"
)

func TestCalcQuotaUsed(t *testing.T) {
	tests := []struct {
		quotaTotal, quotaAddition, quotaLeft, quotaRefund, quotaUsed uint64
		err                                                          error
	}{
		{15000, 5000, 10001, 0, 0, nil},
		{15000, 5000, 9999, 0, 1, nil},
		{10000, 0, 9999, 0, 1, nil},
		{10000, 0, 5000, 1000, 4000, nil},
		{10000, 0, 5000, 5000, 2500, nil},
		{15000, 5000, 5000, 5000, 2500, nil},
		{15000, 5000, 10001, 0, 10000, ErrOutOfQuota},
		{15000, 5000, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 9999, 0, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 1000, 10000, ErrOutOfQuota},
		{10000, 0, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 5000, 5000, 10000, ErrOutOfQuota},
		{15000, 5000, 10001, 0, 0, errors.New("")},
		{15000, 5000, 9999, 0, 1, errors.New("")},
		{10000, 0, 9999, 0, 1, errors.New("")},
		{10000, 0, 5000, 1000, 5000, errors.New("")},
		{15000, 5000, 5000, 5000, 5000, errors.New("")},
	}
	for i, test := range tests {
		quotaUsed := CalcQuotaUsed(test.quotaTotal, test.quotaAddition, test.quotaLeft, test.quotaRefund, test.err)
		if quotaUsed != test.quotaUsed {
			t.Fatalf("%v th calculate quota used failed, expected %v, got %v", i, test.quotaUsed, quotaUsed)
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
