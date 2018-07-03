package main

import "time"

func loop() {
	//d := time.NewTimer(500)
	d1 := time.NewTimer(1000)

	//if !d.Stop() {
	//	c := <-d.C
	//	println("**" + c.String())
	//}
	if !d1.Stop() {
		c := <-d1.C
		println("##" + c.String())
	}
	for {
		select {
		//case c := <-d.C:
		//	{
		//		println("ss*" + c.String())
		//		d.Reset(1000)
		//	}
		case c := <-d1.C:
			{
				println("ss#" + c.String())
				d1.Reset(1000)
			}

		}

	}
}
func main() {
	aChan := make(chan int, 1)
	go loop()
	<-aChan
}
