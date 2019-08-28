# godbg
Start a new project, the debugger on `linux` platform for go   

开个新坑，go语言在`linux`下的调试器    


```
export GO111MODULE=on  
go build -o godbg main.go 
./godbg debug ./test_file/t1.go

or you can `make install` and use `godbg` globally   
```

display  
![display.gif](https://github.com/chainhelen/godbg/blob/master/file/display.gif) 



This project is inspired by [dlv](https://github.com/derekparker/delve) 

> Reference： 
>
> [Writing a Linux Debugger](https://blog.tartanllama.xyz/writing-a-linux-debugger-setup/) 
>
> [elf101-64.pdf](<https://github.com/chainhelen/godbg/blob/master/file/elf101-64.pdf>)
>
> [ELF_Format.pdf](<https://github.com/chainhelen/godbg/blob/master/file/ELF_Format.pdf>)

