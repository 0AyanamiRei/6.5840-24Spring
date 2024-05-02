package main

import (
	"fmt"
	"os"
)

// go run mygo.go BanG_*.txt

func main() {
	fmt.Println(len(os.Args))
	fmt.Println("args[1]=", os.Args[1])
}
