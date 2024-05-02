# 解读函数

## wc.go

`wc.go`文件定义了**MapReduce**程序的`Map`和`Reduce`函数:

1. `Map(filename, contents)`, 两个参数分别是`文件名`和`文件内容`, 然后将文件内容分割成单词, 每个单词生成一个键值对:`<word, 1>`

2. `Reduce(key, values[])`, 第一个参数key即为一个单词, 然后values列表在这里实际就是[1,1,1...], 其长度就代表了这个单词出现在文件中的次数.

我们来具体看看代码,主要是想了解下不同文件之间传递了哪些数据结构

```go
// worker.go中
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

`mr.KeyValue`就是上面在文件`worker.go`中定义的结构体, `Map`函数把`contents`中的每一个单词处理成`<word, 1>`键值对存放在`kva`中,作为函数的返回值返回.

```go
func Reduce(key string, values []string) string {
	return strconv.Itoa(len(values))
}
```

统计一个单词, 也就是`key`在文本中出现的次数, 作为字符串形式返回其出现的次数.

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

其中`pg-*.txt`是一个shell通配符, 匹配当前目录下所有以`pg-`开头的文件, `wc.so`则是一个**plugin**: 定义了**MapReduce**中`map()`和`reduce()`函数的具体内容, 在这里被`loadPlugin(os.Args[1])`从中加载具体的Map和Reduce函数, 返回`mapf`和`reducef`, 也就是这两个函数.

```go
intermediate := []mr.KeyValue{}
for _, filename := range os.Args[2:] {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))
	intermediate = append(intermediate, kva...)
}
```

需要在意的是`kva...`这里的语法, 这是`切片展开操作`, 将kva的内容逐个追加,而不是将kva作为一个元素追加到`intermediate`中; 这里结束后`intermediate`中记录的内容如下样式:

```
[
	{"hello", "1"},
	{"world", "1"},
	{"goodbye", "1"},
	{"world", "1"}
]
```

```go
sort.Sort(ByKey(intermediate))
oname := "mr-out-0"
ofile, _ := os.Create(oname)
```

接下来的操作和真正的MapReduce有很大的区别, 真正的MapReduce中, intermediate的数据会被分区到N*M个桶中, 而这里只是简单的存储在一个切片中.

然后对intermediate切片进行按key排序, 然后创建一个叫`mr-out-0`的文件, 然后返回一个文件句柄`ofile`, 准备进行reduce tasks

```go
i := 0
for i < len(intermediate) {
	j := i + 1
	for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
		j++
	}
	values := []string{}
	for k := i; k < j; k++ {
		values = append(values, intermediate[k].Value)
	}
	output := reducef(intermediate[i].Key, values)

	// this is the correct format for each line of Reduce output.
	fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

	i = j
}
ofile.Close()
```

对intermediate中每一个key(单词)都调用reduce函数, 不过在这之前要把传给reduce的参数处理出来:`key`和`values`, 举个例子就容易理解了:

```
// 排序后
intermediate = [
	{"Sakiko", "1"},
	{"Sakiko", "1"},
	{"Uika", "1"},
	{"Uika", "1"}
]
// 因为已经是排好序了, 所以文件中多次出现的单词产生的键值对会相邻
key="Sakiko"
values = [
	{"1"},
	{"1"}
]
```

将reduce后的结果按照正确的格式写入到文件句柄`ofile`管理的文件中.

至此, 我们了解了一个简单的**MapReduce**的程序是如何运行的, 可以看到它仅仅是一个进程, 没有并行以及分布式的运算, 我们的任务就是改进它!