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
	"fmt"
	"time"
)

func get_i(i int) int {
	return i
}

func func1(y chan string, x chan int) {
	y <- "4"
	<-x
}

func main() {
	x := make(chan int, 0)
	y := make(chan string, 0)

	a := make(chan int, 1)
	b := make(chan int, 0)
	c := make(chan int, 0)
	d := make(chan int, 0)

	go func(c int, d string) { x <- 1; a <- 1; fmt.Println(c, d) }(3, "a")
	go func() { <-x; x <- 1; b <- get_i(1) }()
	go func1(y, x)
	go func() { i := <-y; d <- 6; println(i) }()

	select {
	case <-a:
		println("a")
	case <-b:
		println("b")
	case <-c:
		println("c")
	case <-d:
		println("d")
	default:
		println("default")
	}

	time.Sleep(2 * time.Second)
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
	"fmt"
	"time"
	"github.com/ErikKassubek/GoChan/tracer"
)

func get_i(args_MRAjWwhTHc ...any,) int {
	i := args_MRAjWwhTHc[0].(int)
	return i
}

func func1(args_tcuAxhxKQF ...any,) {
	y := args_tcuAxhxKQF[0].(*tracer.Chan[string])
	x := args_tcuAxhxKQF[1].(*tracer.Chan[int])
	y.Send("4")
	x.Receive()

}

func main() {
	tracer.Init()

	x := tracer.NewChan[int](0)
	y := tracer.NewChan[string](0)

	a := tracer.NewChan[int](1)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[int](0)
	d := tracer.NewChan[int](0)

	tracer.Spawn(func(args_DaFpLSjFbc ...any) {
		var c int = args_DaFpLSjFbc[0].(int)
		var d string = args_DaFpLSjFbc[1].(string)
		x.Send(1)
		a.Send(1)
		fmt.Println(c, d)
	}, 3, "a")

	tracer.Spawn(func(args_XoEFfRsWxP ...any) { x.Receive(); x.Send(1); b.Send(get_i(1)) })
	tracer.Spawn(func1, &y, &x)
	tracer.Spawn(func(args_LDnJObCsNV ...any) { i := y.Receive(); d.Send(6); println(i) })
	
  {
		tracer.PreSelect(true, a.GetId(), b.GetId(), c.GetId(), d.GetId())

		select {
		case <-a.GetChan():
			println("a")
		case <-b.GetChan():
			println("b")
		case <-c.GetChan():
			println("c")
		case <-d.GetChan():
			println("d")
		default:
			println("default")
			tracer.PostDefault()
		}
	}

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
[signal(2), signal(3), signal(4), signal(5), pre(3?, 4?, 5?, 6?, default), post(default)]
[wait(2), pre(1!), post(2, 1, 1!), pre(3!), post(2, 2, 3!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!), pre(4!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?)]
[wait(5), pre(2?), post(4, 1, 2?), pre(6!)]
```