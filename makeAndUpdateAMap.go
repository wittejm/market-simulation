package main

import "fmt"

func makeAndUpdateAMap() {
	myMap := map[string]*int{
		"key": func() *int { v := 1; return &v }(),
	}
	fmt.Printf("val: %d\n", *myMap["key"])
	myMap["key2"] = func() *int { v := 2; return &v }()

	fmt.Printf("key2 val: %d\n", *myMap["key2"])
	*myMap["key2"]++
	fmt.Printf("incremented val of key2: %d\n", *myMap["key2"])
}
