package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

type Coordinator struct {
	nReduce int
	// Your definitions here.
	fileWait  map[string]*TaskInfo
	lock      sync.Mutex
	MapTaskId int

	// reduceId -> MapId
	ReduceWait map[int]*ReduceInfo
}
type TaskInfo struct {
	status int
}

type ReduceInfo struct {
	MapIds []int
	status int
}

const (
	waiting = iota
	processing
	end
)

// 中间文件的合理命名约定是 mr-X-Y，其中 X 是map任务编号，Y 是reduce任务编号

func (c *Coordinator) GetMapTask(args *MapTaskArgs, reply *MapTaskReply) error {
	c.lock.Lock()
	for s, info := range c.fileWait {
		if info.status == waiting {
			reply.File = s
			reply.TaskId = c.MapTaskId
			reply.NReduce = c.nReduce
			info.status = processing
			c.MapTaskId++
			break
		}
	}
	if reply.File == "" {
		reply.StartReduce = len(c.fileWait) == 0
	}
	c.lock.Unlock()

	if reply.File != "" {
		go func(file string) {
			time.Sleep(6 * time.Second)
			c.lock.Lock()
			if c.fileWait[file] != nil {
				c.fileWait[file].status = waiting
			}
			c.lock.Unlock()
		}(reply.File)
	}

	return nil
}

func (c *Coordinator) CompleteMapTask(args *MapTaskArgs, reply *MapTaskReply) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	//延迟的情况 此任务取消了
	if c.fileWait[args.File] == nil {
		return nil
	}
	delete(c.fileWait, args.File)
	for _, i := range args.Reduce {
		if c.ReduceWait[i] == nil {
			c.ReduceWait[i] = &ReduceInfo{MapIds: []int{}}
		}
		rw := c.ReduceWait[i]
		rw.MapIds = append(rw.MapIds, args.MapId)
	}

	return nil
}

func (c *Coordinator) GetReduceTask(args *ReduceTaskArgs, reply *ReduceTaskReply) error {
	c.lock.Lock()
	if len(c.ReduceWait) == 0 {
		reply.end = true
	}
	reply.ReduceId = -1
	for reduceId, info := range c.ReduceWait {
		if info.status == waiting {
			reply.MapIds = info.MapIds
			reply.ReduceId = reduceId
			info.status = processing
			break
		}
	}
	c.lock.Unlock()

	if reply.ReduceId != -1 {
		go func(id int) {
			time.Sleep(6 * time.Second)
			c.lock.Lock()
			if c.ReduceWait[id] != nil {
				c.ReduceWait[id].status = waiting
			}
			c.lock.Unlock()
		}(reply.ReduceId)
	}
	return nil
}
func (c *Coordinator) CompleteReduceTask(args *ReduceTaskArgs, reply *ReduceTaskReply) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.ReduceWait[args.ReduceId] == nil {
		return nil
	}
	delete(c.ReduceWait, args.ReduceId)

	return nil
}

//
// start a thread that listens for RPCs from worker.go
// 启动unix-domain socket的http监听 负责rpc
// rpc调用实际上就是rpc.call("Coordinator.${Coordinator的方法名}",请求体指针，响应体指针)
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// Done
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	c.lock.Lock()
	res := len(c.fileWait) == 0 && len(c.ReduceWait) == 0
	c.lock.Unlock()
	return res
}

// MakeCoordinator
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	fileStatus := map[string]*TaskInfo{}
	for i := range files {
		fileStatus[files[i]] = &TaskInfo{status: waiting}
	}
	c := Coordinator{fileWait: fileStatus, lock: sync.Mutex{}, ReduceWait: map[int]*ReduceInfo{}, nReduce: nReduce}

	c.server()
	return &c
}
