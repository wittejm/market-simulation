package main

import (
	"fmt"
	"time"
)

func SelectOnTwoChannels() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		time.Sleep(time.Second)
		//ch1 <- 0
	}()
	go func() {
		time.Sleep(2 * time.Second)
		ch2 <- 0
	}()
	go func() {
		time.Sleep(3 * time.Second)
		ch2 <- 1
	}()

	select {
	case <-ch1:
		fmt.Println("received on ch1")
	case <-ch2:
		fmt.Println("received on ch2")

	}

	res := <-ch2
	fmt.Println("res", res)
	time.Sleep(time.Second * 3)
}
