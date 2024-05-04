package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type KeyValue struct {
	Key   string
	Value string
}

func createFile(intermediate []KeyValue) string {
	MapNumber := 1
	ReduceNumber := 2
	filename := fmt.Sprintf("mr-%d-%d", MapNumber, ReduceNumber)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("cannot create %v", filename)
	}
	enc := json.NewEncoder(file)
	for _, kv := range intermediate {
		err := enc.Encode(&kv)
		if err != nil {
			log.Fatalf("cannot encode %v", kv)
		}
	}
	file.Close()
	return filename
}

func readFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}

	dec := json.NewDecoder(file)
	for {
		var kv KeyValue
		if err := dec.Decode(&kv); err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("cannot decode %v", kv)
		}
		fmt.Println(kv)
	}
}
func main() {
	intermediate := []KeyValue{
		{"1", "2"},
		{"2", "3"},
		{"3", "4"},
	}
	filename := createFile(intermediate)
	readFile(filename)
}
