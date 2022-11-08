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

func func1(x chan int, i int) {
	x <- i
}

func main() {
	x := make(chan int)
	y := make(chan int)

	a := make(chan int, 1)
	b := make(chan int, 0)
	c := make(chan string, 0)

	i := 3

	go func1(x, i)
	go func() { <-x; x <- 1 }()
	go func(w chan int, i int) { y <- 1; <-x; w <- i }(b, i)
	go func() { <-y }()

	time.Sleep(1 * time.Second)

	select {
	case <-a:
		println("a")
	case <-b:
		println("b")
	case <-c:
		println("c")
	default:
		print("default")
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
	"time"
	"github.com/ErikKassubek/GoChan/tracer"
)

func func1(gochanTracerArg ...any,) {
	x := gochanTracerArg[0].(*tracer.Chan[int])
	i := gochanTracerArg[1].(int)
	x.Send(i)

}

func main() {
	tracer.Init()
	
	x := tracer.NewChan[int](0)
	y := tracer.NewChan[int](0)

	a := tracer.NewChan[int](1)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[string](0)

	i := 3
	
	tracer.Spawn(func1, &x, i)
	
	tracer.Spawn(func(gochanTracerArg ...any) { x.Receive(); x.Send(1) })
	
	tracer.Spawn(func(gochanTracerArg ...any) {
		var w *tracer.Chan[int] = gochanTracerArg[0].(*tracer.Chan[int])
		var i int = gochanTracerArg[1].(int)
		y.Send(1)
		x.Receive()
		w.Send(i)
	}, &b, i)
	
	tracer.Spawn(func(gochanTracerArg ...any) { y.Receive() })

	time.Sleep(1 * time.Second)
	
	{
		tracer.PreSelect(true, a.GetId(), b.GetId(), c.GetId())

		select {
		case <-a.GetChan():
			a.PostSelect()
			println("a")
		case <-b.GetChan():
			b.PostSelect()
			println("b")
		case <-c.GetChan():
			c.PostSelect()
			println("c")
		default:
			tracer.PostDefault()
			print("default")
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
[signal(2), signal(3), signal(4), signal(5), pre(3?, 4?, 5?, default), post(4, 3, 4?)]
[wait(2), pre(1!), post(2, 1, 1!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?), pre(4!), post(4, 3, 4!)]
[wait(5), pre(2?), post(4, 1, 2?)]
```