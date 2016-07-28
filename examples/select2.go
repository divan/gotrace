package main

import (
	"fmt"
	"os"
	"runtime/trace"
	"time"
)

func main() {
	trace.Start(os.Stderr)
	ch, ch1 := make(chan int), make(chan int)
	out := make(chan int)
	go func() {
		for i := 0; i < 20; i++ {
			v := <-out
			fmt.Println("Recv ", v)
			time.Sleep(100 * time.Microsecond)
		}
	}()
	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
		}
	}()
	go func() {
		for i := 0; i < 10; i++ {
			ch1 <- i
		}
	}()

	for i := 0; i < 20; i++ {
		select {
		case v := <-ch:
			out <- v
		case v := <-ch1:
			out <- v
		}
	}
	trace.Stop()
}
