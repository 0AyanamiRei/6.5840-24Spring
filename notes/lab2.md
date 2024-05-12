# Key/Value Server

## 任务要求

本次实验构建一个`key/value`服务器, 确保在网络故障的情况每个操作也能够准确地执行一次, 并且这些操作是*linearizable*. 服务器维护一个`<key, value>`的内存映射, `key`和`value`的变量类型均为`String`

客户端(*Clients*)可以向该服务器发送三种rpc:

1. `Put(key, value)`: *install*或*replace*参数对应键的值
2. `Append(key, arg)`: 将参数`arg`的内容追加到参数指定的键的值, 并返回其旧值
3. `Get(key)`: 获取参数指定键的值

这里是一些补充说明:

```
对不存在的key, Get(key)返回空字符串;
对不存在的key, Append(key, arg)同样返回空字符串, 这意味着假设存在<key, "">
```

每个客户端通过一个使用`Put/Append/Get`三个通信函数的**Clerk**与键值服务器通讯, **Clerk**负责管理这些rpc与服务器的交互.

如果客户端的请求不是并发的, 那么每个客户端应该观察到此前的数据修改, 即严格线性的.
如果客户端的请求是并发的, 返回值与最终状态必须是可线性化的, 即与按一定顺序执行这些操作的结果和状态相同

关于线性化, 在前面的阅读材料中已经学习过了, 同时还了解了是如何测评的.

## TASK1

You'll need to add RPC-sending code to the Clerk `Put/Append/Get` methods in client.go, and implement `Put()`, `Append()` and `Get()` RPC handlers in server.go.

实现*Clerk*的`Put/Append/Get` methods, 在文件`client.go`中, 并且在`server.go`中实现其处理函数

---
刚开始写的时候还是不要对测评器有太多知识, 做完再去阅读测评器的代码,学习一下如何测评并发程序也是很好的.

`client.go`中的函数:

```go
func : {
  MakeClerk()

  // RPC
  Get()
  PutAppend()

  // 调用PutAppend()的 ps:不知道为什么要这样中间多一层
  Put()
  Append()
}
```

`server.go`中的函数:

```go
func : {
  // debug
  DPrintf()

  StartKVServer()

  // RPC handler
  Get()
  Put()
  Append()
}
```

全部实现后可以通过前面两个测试.

## TASK2

先从问题开始, 面对rpc与机器的不可靠性, 我们需要去处理*msg*丢失的情况, 因为超时会导致rpc的函数返回false, 让我们多次发送某个rpc请求, 服务器必须要处理这样的多rpc请求, 以避免重复的添加和修改数据.

也就是说, 问题变成了我们需要识别在客户端中同一次RPC调用的多个call, 并且在处理结束后过滤掉其余的call

一些建议:

```
你需要为客户端的操作唯一标识, 确保服务器对每个操作只进行一次;

你需要仔细考虑服务器应该维护什么状态来过滤重复的操作;

你可以假设一个客户端一次只进行一个操作, 这意味着当客户端发起一个新的RPC时, 上一个RPC已经结束.

性能测试:

go test -run TestMemGet2 > mem1.txt; go test -run TestMemPut2 > mem2.txt ; go test -run TestMemAppend2 > mem3.txt ; go test -run TestMemPutManyClients > mem4.txt ; go test -run TestMemGetManyClients > mem5.txt ; go test -run TestMemManyAppends > mem6.txt
```

让我们重新来设计, 以此保持幂等RPC

### 方案1

为每个客户端请求的*Put*/*Get*/*Append*操作调用`nrand()`分配一个标号, 在服务器会为每次操作缓存其返回值,直到客户端确认收到服务器的回复为止, 以客户端请求`Get()`为例:

```
          Get_c()
Client-------------->Server     √
          Get_s() 
Client<--------------Server     ×:ok==false
          Get_s() 
Client<--------------Server     ×:ok==false
          Get_s() 
Client<--------------Server     √
```

在*Server*向*Client*开始发送返回消息到*Client*收到该消息为止, 服务器都必须要缓存客户端此次调用应该看到的返回值.

因为我已经提前试过错了, 测试点检测到内存使用情况, 所以我们必须考虑释放缓存的问题, 关于这一点我现在有两个解决办法:

1. 在客户端确认收到服务器的返回消息后再发送另外一条rpc告诉服务器, 这应该是所谓的2p:2-phase吧
2. 服务器这边开一个线程, 定期清理所有缓存的键值对(在这里缓存的形式是键值对)

***客户端***

客户端为每次请求操作分配一个标号, 并且反复发送rpc消息, 直到成功收到服务器的回复, 以`Get()`为例:

```go
args := GetArgs{
  Key:   key,
  OpReg: nrand(), // 生成标号
}
reply := GetReply{}

// 不断发送请求
for ok := ck.server.Call("KVServer.Get", &args, &reply); !ok; {}
```

需要注意的是, 可能你担心发送的频率是否过高, 但是在我检查后发现, 在这次实验中, 一次操作所重复发送的rpc消息, 最多个位数(2-4)次, 所以不必发送后睡眠一段时间.

***服务器***

我们为每次操作缓存其返回值, 为此需要额外的数据结构`OpCache map[int64]string`

同样以`Get()`为例:

```go
value, ok := kv.OpCache[opReg]

// 滤去重复rpc
if ok {
  reply.Value = value
  return
}

// 缓存此次的rpc返回值
kv.OpCache[opReg] = kv.data[key]
reply.Value = kv.data[key]
```

注意我们现在没有开定期清理, 因为我发现开了定期清理的话, 会出现一个情况(并不少见), 某一个rpc在重复发送的时候清理线程把服务器为其准备的缓存清理掉了, 导致这个操作重复进行了.

如果缓存的数据不清理掉的话, 关于内存的测试是无法通过的, 或者说我觉得去缓存数据本身是一个很蠢的行为, 可以思考一下有无不缓存数据本身同时又能保持幂等的操作.

我们暂时采取另一个解决方法, 即让客户端确切收到消息后向服务器发送消息主动清理掉缓存内容