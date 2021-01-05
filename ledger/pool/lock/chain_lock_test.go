package lock

import "testing"

func TestLock(t *testing.T) {
	impl := &EasyImpl{}
	impl.LockInsert()
	impl.UnLockInsert()
	impl.LockRollback()
	impl.UnLockRollback()
	impl.RLockRollback()
	impl.RLockRollback()
	impl.RUnLockRollback()
	impl.RUnLockRollback()
	impl.RLockInsert()
	impl.RLockInsert()
	impl.RUnLockInsert()
	impl.RUnLockInsert()
}
