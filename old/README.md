# godbg
Start a new project, the debugger on linux platform for go   
Copy part of the code as a personal learning only  

开个新坑，go语言在linux下的调试器    
本项目仅作为个人尝试学习之用，故会复制部分delve代码  

```
export GO111MODULE=on  
go build -o main main.go   

// if you want to output log of debug, please do like below(如果想开调试日志，需要如下操作)  
// export DBGLOG=stdout  

./main debug ./test.go  
```


inspire by [dlv](https://github.com/derekparker/delve)  
