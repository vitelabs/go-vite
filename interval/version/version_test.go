package version

import (
	"testing"
	"time"
)

func TestForkVersion(t *testing.T) {
	version := ForkVersion()
	if version != 0 {
		t.Error("expect for 0.")
	}

	IncForkVersion()
	IncForkVersion()
	IncForkVersion()
	version = ForkVersion()
	if version != 3 {
		t.Error("expect for 3.")
	}

	for i := 0; i < 100; i++ {
		go func() {
			IncForkVersion()
		}()
	}
	time.Sleep(time.Second * 2)
	version = ForkVersion()
	if version != 103 {
		t.Error("expect for 103.")
	}
}
