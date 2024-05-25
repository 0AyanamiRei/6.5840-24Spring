package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	p "test/pack"
	"time"
)

type FILE struct {
	Context  map[string]bool
	Filename []string
	mu       sync.Mutex
}

func (t *FILE) readFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		_, ok := t.Context[scanner.Text()]
		if !ok {
			t.Context[scanner.Text()] = true
		}
	}

	file.Close()

}

func (t *FILE) GetFilename(dirname string) {
	files, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatal(err)
	}

	t.mu.Lock()
	for _, file := range files {
		// 模拟进行读取
		ms := 200 + (rand.Int63() % 200)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		t.Filename = append(t.Filename, dirname+file.Name())
	}
	t.mu.Unlock()
}

func (t *FILE) ReadDirfile() {
	dirname := "D:/Project/6828/6.5840-24Spring/test/data/"
	t.GetFilename(dirname)
}

func (t *FILE) ReadFile() {
	for _, name := range t.Filename {
		go func(fname string) {
			// 模拟进行读取
			ms := 200 + (rand.Int63() % 200)
			time.Sleep(time.Duration(ms) * time.Millisecond)
			t.readFile(fname)
		}(name)
	}
}

func main() {
	t := FILE{
		Context:  map[string]bool{},
		Filename: make([]string, 0),
	}

	go func() {
		p.DPrintf("开始扫描目录\n")
		t.ReadDirfile()
	}()

	for k := range t.Context {
		fmt.Println(k)
	}
}
