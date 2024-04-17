package main

import "fmt"

type Obj struct {
	field int
}

func updateTheValueOfAPointer() {
	myObj := Obj{
		1,
	}
	fmt.Println(myObj)
	myInt := (&myObj).field
	fmt.Println("int from at field of pointer:", myInt)
	myObjPointer := &myObj
	myIntPointer := &myObjPointer.field
	fmt.Println("int pointer:", myIntPointer)
	fmt.Println("int val from pointer of int:", *myIntPointer)
	fmt.Println("incrementing via the int pointer...")
	*myIntPointer++
	fmt.Println("int val from pointer of int:", *myIntPointer)
	fmt.Println("original int:", myInt)
	fmt.Println("the value of the object's field", myObj.field)
	fmt.Println("the value of the object's field via the object pointer", (&myObj).field)

}
