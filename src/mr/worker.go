package mr

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

loop:
	args := MapTaskArgs{}
	reply := MapTaskReply{}
	call("Coordinator.GetMapTask", &args, &reply)
	if reply.File != "" {
		args.File = reply.File
		args.Reduce = mapTask(reply.File, reply.TaskId, reply.NReduce, mapf)
		args.MapId = reply.TaskId
		call("Coordinator.CompleteMapTask", &args, &reply)
		goto loop
	} else if !reply.StartReduce {
		time.Sleep(5 * time.Millisecond)
		goto loop
	}

reduceLoop:

	reduceArgs := ReduceTaskArgs{}
	reduceReply := ReduceTaskReply{}
	call("Coordinator.GetReduceTask", &reduceArgs, &reduceReply)
	if !reduceReply.end {
		if reduceReply.ReduceId != -1 {
			reduceTask(reduceReply.MapIds, reduceReply.ReduceId, reducef)
			reduceArgs.ReduceId = reduceReply.ReduceId
			call("Coordinator.CompleteReduceTask", &reduceArgs, &reduceReply)
		} else {
			time.Sleep(5 * time.Millisecond)
		}
		goto reduceLoop
	}
}
func mapTask(filename string, mapTaskId int, n int, mapf func(string, string) []KeyValue) []int {
	reduceIds := []int{}
	reduceFile := map[int]*os.File{}
	file, _ := os.Open(filename)
	content, _ := ioutil.ReadAll(file)
	file.Close()
	values := mapf(filename, string(content))
	for _, value := range values {
		reduce := ihash(value.Key) % n
		if reduceFile[reduce] == nil {
			reduceFile[reduce], _ = os.Create(fmt.Sprintf("mr-%d-%d", mapTaskId, reduce))
		}
		fmt.Fprintf(reduceFile[reduce], "%v %v\n", value.Key, value.Value)
	}
	for r, o := range reduceFile {
		o.Close()
		reduceIds = append(reduceIds, r)
	}
	return reduceIds
}
func reduceTask(mapIds []int, reduceId int, reducef func(string, []string) string) {
	open, _ := os.Create(fmt.Sprintf("mr-out-%d-%d", reduceId, rand.Intn(1000000)))
	defer open.Close()
	intermediate := []KeyValue{}
	for _, id := range mapIds {
		file, _ := os.Open(fmt.Sprintf("mr-%d-%d", id, reduceId))
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var key, value string
			fmt.Sscanf(scanner.Text(), "%v %v", &key, &value)
			intermediate = append(intermediate, KeyValue{Key: key, Value: value})
		}
		file.Close()
	}
	sort.Sort(ByKey(intermediate))
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

		fmt.Fprintf(open, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	os.Rename(open.Name(), fmt.Sprintf("mr-out-%d", reduceId))

}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
