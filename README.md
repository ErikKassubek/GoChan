# GoChan: Dynamic Analysis of Message Passing Go Programms

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

Witten elaboration at: https://github.com/ErikKassubek/Bachelorarbeit

## Makefile 
You can run the instrumenter and build the created output by running
```
make IN="<input>"
```
where \<input> is the path to the code.

## Example
We use the following go code as example:
```
// ./fold/main.go
package main

import (
	"time"
)

// import "time"

/*
func main() {
	tracer.Init()

	x := tracer.NewChan[int](0)
	y := tracer.NewChan[string](0)

	a := tracer.NewChan[int](0)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[int](0)
	d := tracer.NewChan[int](0)

	tracer.Spawn(func() { x.Send(1); a.Send(0) })
	tracer.Spawn(func() { x.Receive(); x.Send(2); b.Send(0) })
	tracer.Spawn(func() { y.Send("a"); x.Receive(); c.Send(0) })
	tracer.Spawn(func() { y.Receive(); d.Send(0) })

	select {
	case <-a.GetChan():
		println("a")
	case <-b.GetChan():
		println("b")
	case <-c.GetChan():
		println("c")
	case <-d.GetChan():
		println("d")
	}

	time.Sleep(2 * time.Second)
	tracer.PrintTrace()
}
*/

/*
func run() {
	fmt.Println("run")
}

func runArg(args ...any) {
	var a int = args[0].(int)
	var b string = args[1].(string)
	fmt.Println("runArg: ", a, b)
}

func runArgs(args1 ...any) {
	var i int = args1[0].(int)
	var args []any = args1[1:]
	fmt.Print("runArgs: ", i, " ")
	for _, i := range args {
		fmt.Print(i, " ")
	}
}
*/

/*
func get_i(i int) int {
	return i
}

func run() <-chan struct{} {

}

func runArg(a int, b string, c int) {
	fmt.Println("runArg: ", a, b, c)
}

func runArgs(i int, s ...string) {
	fmt.Print("runArgs: ", i, " ")
	for _, i := range s {
		fmt.Print(i, " ")
	}
	fmt.Println("")
}

func func_in_func(f func(i int)) {}

func main() {
	func_in_func(func(x int) {})
	ex := []any{1, 2, 3}
	var args []int
	for _, a := range ex[1:] {
		args = append(args, a.(int))
	}
	fmt.Println(args)

	i := 0

	x := make(chan int, 0)
	y := make(chan string, 0)

	a := make(chan int, 0)
	b := make(chan int, 0)
	c := make(chan int, 0)
	d := make(chan int, 0)

	// go run()
	// go runArg(1, "a", 1)
	// go runArgs(1, "a", "b")
	go func(c int, d string) { x <- i; a <- 1; fmt.Println(c, d) }(3, "a")
	go func() { <-x; x <- 1; b <- get_i(1) }()
	go func() { y <- "4"; <-x; c <- 5 }()
	go func() { i := <-y; d <- 6; ; println(i) }()

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
*/

/*
func get_chan(x int) chan int {
	c := make(chan int)
	return c
}

func main() {
	a := make(chan int)
	b := make(chan int)

	select {
	case <-a:
		fmt.Println("1")
	case x := <-b:
		fmt.Println(x)
	case <-get_chan(3):
		fmt.Println("c")
		// case y := <-get_chan():
		// 	fmt.Println(y)
	}
}
*/

// func get_i(i int) int {
// 	return i
// }

// func func1(y chan string, x chan int) {
// 	y <- "1"
// 	<-x
// }

func func1(x chan int, i int) {
	x <- 1
}

func main() {
	x := make(chan int)
	y := make(chan int)

	a := make(chan int, 1)
	b := make(chan int, 0)
	c := make(chan string, 0)

	i := 3

	go func1(x, i)
	go func() {
		<-x
		x <- 1
	}()
	go func(i int) {
		y <- 1
		<-x
	}(i)
	go func() { <-y }()

	time.Sleep(1 * time.Second)

	select {
	case x := <-a:
		println(x)
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
```./instrumenter/instrumenter -chan -in=./fold -show_trace```
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

func func1(x tracer.Chan[int], i int) {
	x.Send(1)

}

func main() {
	tracer.Init()
	x := tracer.NewChan[int](0)
	y := tracer.NewChan[int](0)

	a := tracer.NewChan[int](1)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[string](0)

	i := 3
	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{

				func1(x, i)
			}
		}()
	}()
	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				x.Receive()
				x.Send(1)
			}
		}()
	}()
	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func(i int) {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				y.Send(1)
				x.Receive()
			}
		}(i)
	}()
	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				y.Receive()
			}
		}()
	}()

	time.Sleep(1 * time.Second)
	{
		tracer.PreSelect(true, b.GetId(), c.GetId())

		select {
		case x_sel := <-a.GetChan():
			x := x_sel.GetInfo()
			a.PostSelect()
			println(x)
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
[signal(2), signal(3), signal(4), signal(5), pre(4?, 5?, default), post(default)]
[wait(2), pre(1!), post(2, 1, 1!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?)]
[wait(5), pre(2?), post(4, 1, 2?)]
```