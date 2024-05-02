# 解读函数

## wc.go

`wc.go`文件定义了**MapReduce**程序的`Map`和`Reduce`函数:

1. `Map(filename, contents)`, 两个参数分别是`文件名`和`文件内容`, 然后将文件内容分割成单词, 每个单词生成一个键值对:`<word, 1>`

2. `Reduce(key, values[])`, 第一个参数key即为一个单词, 然后values列表在这里实际就是[1,1,1...], 其长度就代表了这个单词出现在文件中的次数.

我们来具体看看代码,主要是想了解下不同文件之间传递了哪些数据结构.

```go
// worker.go
type KeyValue struct {
	Key   string
	Value string
}

func Map(filename string, contents string) []mr.KeyValue {

	ff := func(r rune) bool { return !unicode.IsLetter(r) }

	words := strings.FieldsFunc(contents, ff)
	kva := []mr.KeyValue{}

	for _, w := range words {
		kv := mr.KeyValue{w, "1"}
		kva = append(kva, kv)
	}

	return kva
}
```


## mrsequential.go

```go
if len(os.Args) < 3 {
  fmt.Fprintf(os.Stderr, "Usage: mrsequential xxx.so inputfiles...\n")
  os.Exit(1)
}
mapf, reducef := loadPlugin(os.Args[1])
```

`os.Args`, 这是存储`go run xxx传入的参数`的string数组, 示范代码中输入了:

```sh
go run mrsequential.go wc.so pg*.txt
```

其中`pg-*.txt`是一个shell通配符, 匹配当前目录下所有以`pg-`开头的文件.