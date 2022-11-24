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
	"math/rand"
	"sync"
	"time"
)

func func1(x chan int) {
	x <- rand.Intn(100)
}

func test() {

	x := make(chan int)
	y := make(chan int)

	a := make(chan int, 1)
	b := make(chan int, 0)
	c := make(chan string, 0)

	var l sync.Mutex
	var m sync.RWMutex

	i := 3

	go func1(x)
	go func() {
		m.Lock()
		<-x
		x <- rand.Intn(100)
		m.Unlock()
	}()
	go func(i int) {
		l.Lock()
		y <- i
		<-x
		l.Unlock()
	}(i)
	go func() {
		m.RLock()
		<-y
		m.RUnlock()
	}()

	time.Sleep(1 * time.Second)

	select {
	case x := <-a:
		println(x)
	case <-b:
		println("b")
	case c <- "3":
		println("c")
	default:
		println("default")
	}
}

func main() {
	test()
	time.Sleep(4 * time.Second)
}
```
After running the makefile with
```make IN="fold/"```
where fold is the folder containing our project, including our main.go file, an output folder is created.
./fold should contains a go.mod and go.sum file.
The output folder contains a folder fold, which contains the translated files.
In our case we get 
```
// ./output/fold/main.go
package main

import (
	"math/rand"
	"time"

	"github.com/ErikKassubek/GoChan/tracer"
)

func func1(x tracer.Chan[int]) {
	x.Send(rand.Intn(100))
}

func test() {

	x := tracer.NewChan[int](0)
	y := tracer.NewChan[int](0)

	a := tracer.NewChan[int](1)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[string](0)

	l := tracer.NewLock()
	m := tracer.NewRWLock()

	i := 3

	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{

				func1(x)
			}
		}()
	}()

	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				m.Lock()
				x.Receive()
				x.Send(rand.Intn(100))
				m.Unlock()
			}
		}()
	}()

	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func(i int) {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				l.Lock()
				y.Send(i)
				x.Receive()

				l.Unlock()
			}
		}(i)
	}()

	func() {
		GoChanRoutineIndex := tracer.SpawnPre()
		go func() {
			tracer.SpawnPost(GoChanRoutineIndex)
			{
				m.RLock()
				y.Receive()

				m.RUnlock()
			}
		}()
	}()

	time.Sleep(1 * time.Second)

	{
		tracer.PreSelect(true, a.GetIdPre(true), b.GetIdPre(true), c.GetIdPre(false))
		sel_HctcuAxh := tracer.BuildMessage("3")

		select {
		case sel_XVlBzgbaiC := <-a.GetChan():
			a.PostSelect(true, sel_XVlBzgbaiC)
			x := sel_XVlBzgbaiC.GetInfo()
			println(x)
		case sel_MRAjWwhT := <-b.GetChan():
			b.PostSelect(true, sel_MRAjWwhT)
			println("b")
		case c.GetChan() <- sel_HctcuAxh:
			c.PostSelect(false, sel_HctcuAxh)
			println("c")
		default:
			tracer.PostDefault()
			println("default")
		}
	}
}

func main() {
	tracer.Init()
	defer tracer.PrintTrace()

	test()

	time.Sleep(4 * time.Second)
}
```
We can now run the translated project and get a trace. One possible trace is
```
[signal(2), signal(3), signal(4), signal(5), pre(3?, 4?, 5!, default), post(default)]
[wait(2), pre(1!), post(2, 1, 1!)]
[wait(3), lock(2, -, 1), pre(1?)]
[wait(4), lock(1, -, 1), pre(2!), post(4, 2, 2!), pre(1?), post(2, 1, 1?), unlock(1)]
[wait(5), lock(2, r, 1), pre(2?), post(4, 2, 2?), unlock(2)]
```
An explenation of the trace can be found in the [tracer](https://github.com/ErikKassubek/GoChan/tree/main/tracer).
