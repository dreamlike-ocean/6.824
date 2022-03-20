## lab1

### 设计点

第一个lab很简单就是 分发map任务，map完成后再分发reduce任务

其设计要点在于：

- 为了容易实现，“分发”这个操作不是由server主动发送的，而是client去请求任务
- test中存在crash_tes会导致部分map或者reduce的时候crash或者timeout，这个时候就要考虑这么几个边界case   1，任务分发完毕并且都在处理，此时还是处于map阶段，新任务请求进来之后不能直接获取到reduce，而是等待信号再去获取reduce  
- 因为存在job_count的test，所以timeout计时器时间最好高于5s
- 定时器不能太高，因为每个测试都存在整体的timeout测试
- 线程安全问题，直接sync.lock即可
- 分发map就是分发文件名，分发reduce就是分发reduce id和对应的mapid
- 提交map任务会随之提交其对应产生的reduceIds []int和与之匹配的mapId,方便定位文件（中间文件的合理命名约定是 mr-X-Y，其中 X 是map任务编号，Y 是reduce任务编号）
- 当获取map/reduce任务成功时，要开启一个协程去做定时任务，若发现对应时间其任务还是没有完成就重置任务状态
- 同时server接收到完成任务提交时也要检查一下是不是任务已经完成了，以应对延时这种情况



## lab2

**请务必去看完完整的lab辅导课 就是 go thread and raft一节**

### Part 2A: leader election ([moderate](https://pdos.csail.mit.edu/6.824/labs/guidance.html))

您的任务是：实现Raft leader选举和心跳（没有log项的`AppendEntries`RPC）。第2A部分的目标是选出一位leader，如果没有出现异常，leader将继续担任；leader，如果旧leader失败或旧leader的数据包丢失，则由新领导人接任。请用`go test -run 2A`来测试你的2A代码

提示：

- 您很难直接运行您实现的raft；因此您应当通过测试器的方式运行它，比如 `go test -run 2A `.
- 按照paper的 Figure2的指示。此时您应当关心发送并接收RequestVote的RPC,与选举相关的服务器准则和与leader选举相关的状态
- 将figure2的leade选举状态添加到`raft.go`的`Raft`结构体中。您将需要定义一个结构体来保存每一个log项的信息
- 填充`RequestVoteArgs`和`RequestVoteReply`结构体，修改`Make()`方法创建一个后台的goroutine，当它有一段时间没有收到其他peer的消息时，通过发送`RequestVote` RPC定期启动领导人选举。这样，如果已经有leader，peer将了解谁是领导者，或者自己成为leader。实现`RequestVote()`RPC处理程序，以便服务器可以相互投票。
- 要实现heartbeats，请定义一个`AppendEntries` RPC结构（尽管您可能还不需要所有参数），并让领导者定期发送它们。编写一个`AppendEntries` RPC handler方法，重置选举定时器，这样其他服务器就不会在一个已经被选举的服务器上作为领导者前进。
- 确保不同peer的选举暂停时间不总是同时发生，否则所有peer只为自己投票，没有人会成为领导人。
- 测试器要求领导者每秒发送心跳RPC的次数不超过10次。
- 测试器要求你的Raft在旧领导失败后的五秒钟内选出新领导（如果大多数peer仍然可以沟通）。但是，请记住，如果出现分裂投票，领导人选举可能需要多轮投票（如果数据包丢失或候选人不幸选择了相同的随机退避时间，则可能会发生这种情况）。你必须选择足够短的选举超时时间（以及心跳间隔），即使需要多轮投票，选举也很可能在不到五秒钟内完成
- paper 第5.2节提到选举暂停时间在150到300毫秒之间。只有当领导者每150毫秒发送一次以上的心跳时，这样的范围才有意义。由于测试员将您的心跳限制为每秒10次，因此您必须使用大于paper中 150到300毫秒的选举超时，但不能太大，因为这样您可能无法在5秒钟内选举领导人
- 您会发现GO的[rand](https://golang.org/pkg/math/rand/) 非常有用
- 您需要编写定时或延时后执行操作的代码。最简单的方法是创建一个goroutine，其中包含一个调用 [time.Sleep()](https://golang.org/pkg/time/#Sleep)循环；（请参见`Make()`为此创建的`ticker()`goroutine）。不要利用GO的时间`time.Timer` or `time.Ticker`, ，这些很难正确使用。
- [Guidance page](https://pdos.csail.mit.edu/6.824/labs/guidance.html)有一些建议可以帮助您开发和调试您的代码
- 如果您的代码无法通过测试，请再次阅读paper的Figure 2；领导人选举的全部逻辑分布在这个figure的多个部分。
- 不要忘记实现`GetState`
- 调试器会调用您Raft内的`rf.Kill()`，以关闭这个实例。您可以检查`Kill()`是否通过使用`rf.killed()`被调用。您或许想在全部的循环内部要做一些事情以避免raft实例打印一些看不懂的消息
- GO RPC只发送以大写开头的结构体内部的字段，子结构体必须具有大写的字段名（比如数组中log recods 对应的字段）。`labgob`包将警告您这一点需要注意，不要忽视waring

在提交第2A部分之前，请确保通过了2A测试，以便看到如下内容：

```shell
$ go test -run 2A
Test (2A): initial election ...
  ... Passed --   3.5  3   58   16840    0
Test (2A): election after network failure ...
  ... Passed --   5.4  3  118   25269    0
Test (2A): multiple elections ...
  ... Passed --   7.3  7  624  138014    0
PASS
ok  	6.824/raft	16.265s
$
```

每个“pass”行包含五个数字；这些是测试所用的时间（以秒为单位）、Raftpeer的数量、测试期间发送的RPC的数量、RPC消息中的总字节数，以及Raft报告提交的日志条目数。您的数字将与此处显示的数字不同。如果愿意，可以忽略这些数字，但它们可以帮助您检查实现发送的RPC的数量。对于所有lab2、3和4，如果所有测试（`go test`）所需时间超过600秒，或者任何单个测试所需时间超过120秒，评分脚本将判定您的答案是错的。

当我们为您提交的内容评分时，我们将在不带-race标志的情况下运行测试，但您还应该确保您的代码始终通过带-race标志的测试。

#### 设计点

1，若发现term比自己高的AppendRequest必须退化为follwer状态

2，**涉及到访问rf内属性，无论是修改还是访问都请加锁**

3，统计票数的时候，不要使用计数为len(peer)-1的waitGroup，试想一个情况，0，1，2。2被分区或者延迟了，2投给了1，而1在不断等待3的回应，此时虽已经满足成为leader的条件，但由于在等待，此时就会导致2超时再起来竞选

4，论文中提到RPC的term大于此时的raft实例的term时，必须要回到foller状态，这个RPC指的是全部的RPC而非心跳RPC，

```go
if rf.term < args.Term {
		rf.becomeFollower(args.Term)
	}
```

**5，跑test前请加上-race 参数**，把自己代码中的race解决，需要锁的时候，请立刻在获取锁下面加`defer lock.unlock()`

6，不要在rpc时持有锁 防止和接收心跳并发时 心跳那边卡住

7，几个常数参考

```go
const HeartBeatInterval = 120
const ElectronInterval = 300
const (
	follower = iota
	candidate
	lead
)
```



### Part 2B: log

您的任务是：实现leader和follower代码以附加新的日志项，以便 `go test -run 2B  `测试通过。

提示：

- 使用`git pull`指令获取最新的lab软件
- 您的第一个目标是通过`TestBasicAgree2B()`.首先实现`Start()`，然后编写代码，通过`AppendEntries` RPC发送和接收新的日志项，如Figure2 所示。在每个peer上通过`applyCh`发送每个新commit的日志项。(译者：`applyCh` raft实例的make方法的那个同名参数)
- 您需要实现paer5.4.1部分的选举限制。
- 在test的前面部分中未能达成一致的一种情况是，即使领导人还活着，也举行了多次选举。请在选举计时器部分找找bug，或者在赢得选举后不立即发送心跳。
- 您的代码可能有重复检查某些事件的循环，不要让这些循环连续执行而不暂停，因为这会大大降低您的实现运行速度，导致测试失败。请使用go的[条件变量](https://golang.org/pkg/sync/#Cond)，或者在每个循环迭代中插入一段`time.Sleep(10 * time.Millisecond)`
- 为未来的lab写起来容易，请帮自己一个忙，写（或重写）出干净清晰的代码，为了实现这个想法请再次访问我们的 [Guidance page](https://pdos.csail.mit.edu/6.824/labs/guidance.html)，来获取一些关于开发或者debug的建议 
- 如果发现没有通过测试，就看看`config.go`和`test_test.go`中的测试代码，这些代码可以帮助你更好理解是怎么测试的。`config.go`也描述了测试器是怎么测试raft api代码的

如果代码运行太慢，lab测试可能会失败。您可以使用time命令检查解决方案使用的实时时间和CPU时间。

以下是典型的输出：

```shell
$ time go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   0.9  3   16    4572    3
Test (2B): RPC byte count ...
  ... Passed --   1.7  3   48  114536   11
Test (2B): agreement after follower reconnects ...
  ... Passed --   3.6  3   78   22131    7
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.8  5  172   40935    3
Test (2B): concurrent Start()s ...
  ... Passed --   1.1  3   24    7379    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   5.1  3  152   37021    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  17.2  5 2080 1587388  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.2  3   60   20119   12
PASS
ok  	6.824/raft	35.557s

real	0m35.899s
user	0m2.556s
sys	0m1.458s
$
```

“ok 6.824/raft 35.557s” 表示Go测量的2B测试所用时间为实际（挂钟）时间的35.557秒。“user 0m2.556s”意味着代码消耗了2.556秒的CPU时间，或实际执行指令所花费的时间（而不是等待或睡眠）。如果您的解决方案在2B测试中使用的实时时间远远超过一分钟，或远远超过5秒的CPU时间，您可能会在以后遇到麻烦。查找花在睡眠或等待RPC超时、不睡眠或等待条件或通道消息的循环，或发送的大量RPC的时间