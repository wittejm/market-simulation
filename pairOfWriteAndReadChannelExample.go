package main

import (
	"fmt"
	"time"
)

// MinimalChannelExample comment
func MinimalChannelExample() {
	chanA := make(chan string)
	chanB := make(chan string)
	go writeThenRead(chanA, chanB, "hello first")
	go readThenWrite(chanA, chanB, "hello second")
	go pubToChan(chanA, "hello direct")
	go readFromChan(chanA)
	time.Sleep(time.Second * 4)

}

func writeThenRead(writeChannel chan string, readChannel chan string, msg string) {
	writeChannel <- msg
	received := <-readChannel
	fmt.Printf("received %s\n", received)
}

func readThenWrite(readChannel chan string, writeChannel chan string, msg string) {
	received := <-readChannel
	writeChannel <- msg
	fmt.Printf("received %s\n", received)
}

func pubToChan(channel chan string, msg string) {
	fmt.Println("Publishing to channel")
	channel <- msg
	fmt.Println("Published to channel")
}

func readFromChan(channel chan string) string {
	fmt.Println("reading from channel")
	msg := <-channel
	fmt.Printf("read from channel. Msg: %s\n", msg)
	return msg
}
