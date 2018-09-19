package compress

var compressor *Compressor

//func TestMain(m *testing.M) {
//	compressor = NewCompressor()
//	os.Exit(m.Run())
//}
//
//func TestCompressor_Start_Stop(t *testing.T) {
//	compressor.Start()
//	time.Sleep(time.Second * 3)
//	compressor.Stop()
//}
//
//func TestCompressor_StartMultiTime(t *testing.T) {
//	compressor.Start()
//	if compressor.Start() {
//		t.Error("Can't start again")
//	}
//	if compressor.Start() {
//		t.Error("Can't start again")
//	}
//}
