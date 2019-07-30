# godbg
Start a new project, the debugger on `linux` platform for go   

开个新坑，go语言在`linux`下的调试器    


```
export GO111MODULE=on  
go build -o godbg main.go   

// if you want to output log of debug, please do like below(如果想开调试日志，需要如下操作)  
// export DBGLOG=stdout  

./godbg debug ./test.go  
```

inspire by [dlv](https://github.com/derekparker/delve)  (has removed codes that copied from `dlv`)

> Reference： 
>
> [Writing a Linux Debugger](https://blog.tartanllama.xyz/writing-a-linux-debugger-setup/) 
>
> [elf101-64.pdf](<https://github.com/chainhelen/godbg/blob/master/file/elf101-64.pdf>)
>
> [ELF_Format.pdf](<https://github.com/chainhelen/godbg/blob/master/file/ELF_Format.pdf>)

