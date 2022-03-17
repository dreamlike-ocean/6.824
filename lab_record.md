## lab1

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

