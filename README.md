# GoChan: Dynamic Analysis of Message Passing Go Programms

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## Example
We use the following go code as example:
```
// ./fold/main.go

package main

import (
	"time"
)

func main() {
	x := make(chan int, 0)
	y := make(chan int, 0)

	go func() { x <- 1 }()
	go func() { <-x; x <- 1 }()
	go func() { y <- 1; <-x }()
	go func() { <-y }()

	time.Sleep(2 * time.Second)
}
}
```
After running the instrumenter with
```./instrumenter/instrumenter -in=./fold -show_trace```
where ./fold is the folder containing our project, including our main.go file, an output folder is created.
./fold should contains a go.mod and go.sum file.
The output folder contains a folder ./fold, which contains the translated files.
In our case we get 
```
// ./output/fold/main.go
package main

import (
	"time"
	"github.com/ErikKassubek/GoChan/tracer"
)

func main() {
	tracer.Init()
	x := tracer.NewChan[int](0)
	y := tracer.NewChan[int](0)
	tracer.Spawn(func(args_MRAjWwhTHc ...any) { x.Send(1) })
	tracer.Spawn(func(args_tcuAxhxKQF ...any) { x.Receive(); x.Send(1) })
	tracer.Spawn(func(args_DaFpLSjFbc ...any) { y.Send(1); x.Receive() })
	tracer.Spawn(func(args_XoEFfRsWxP ...any) { y.Receive() })

	time.Sleep(2 * time.Second)
	tracer.PrintTrace()
}
```
After installing the tracer library with 
``` 
go get github.com/ErikKassubek/GoChan/tracer
```
in ./output/fold/, we can run the translated project and get the following trace:
```
[signal(2), signal(3), signal(4), signal(5)]
[wait(2), pre(1!), post(2, 1, 1!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?)]
[wait(5), pre(2?), post(4, 1, 2?)]
```