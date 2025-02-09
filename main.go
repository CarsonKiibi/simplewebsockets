package main

import (
	"fmt"
	"github.com/carsonkiibi/websockets/server"
)

func main() {
	server.PrintBits()
	number := 0
	anded := number | 0b1111
	fmt.Println(anded)
}
