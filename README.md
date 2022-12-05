# GoChan: Dynamic Analysis of Message Passing Go Programms

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

Written elaboration at: https://github.com/ErikKassubek/Bachelorarbeit

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

	"github.com/ErikKassubek/GoChan/goChan"
)

func func1(x goChan.Chan[int]) {
	x.Send(rand.Intn(100))
}

func test() {

	x := goChan.NewChan[int](0)
	y := tracer.NewChan[int](0)

	a := goChan.NewChan[int](1)
	b := goChan.NewChan[int](0)
	c := tracer.NewChan[string](0)

	l := goChan.NewLock()
	m := goChan.NewRWLock()

	i := 3

	func() {
		GoChanRoutineIndex := goChan.SpawnPre()
		go func() {
			goChan.SpawnPost(GoChanRoutineIndex)
			{

				func1(x)
			}
		}()
	}()

	func() {
		GoChanRoutineIndex := goChan.SpawnPre()
		go func() {
			goChan.SpawnPost(GoChanRoutineIndex)
			{
				m.Lock()
				x.Receive()
				x.Send(rand.Intn(100))
				m.Unlock()
			}
		}()
	}()

	func() {
		GoChanRoutineIndex := goChan.SpawnPre()
		go func(i int) {
			goChan.SpawnPost(GoChanRoutineIndex)
			{
				l.Lock()
				y.Send(i)
				x.Receive()

				l.Unlock()
			}
		}(i)
	}()

	func() {
		GoChanRoutineIndex := goChan.SpawnPre()
		go func() {
			goChan.SpawnPost(GoChanRoutineIndex)
			{
				m.RLock()
				y.Receive()
				m.RUnlock()
			}
		}()
	}()

	time.Sleep(1 * time.Second)

	{
		goChan.PreSelect(true, a.GetIdPre(true), b.GetIdPre(true), c.GetIdPre(false))
		sel_HctcuAxh := goChan.BuildMessage("3")

		select {
		case sel_XVlBzgbaiC := <-a.GetChan():
			a.Post(true, sel_XVlBzgbaiC)
			x := sel_XVlBzgbaiC.GetInfo()
			println(x)
		case sel_MRAjWwhT := <-b.GetChan():
			b.Post(true, sel_MRAjWwhT)
			println("b")
		case c.GetChan() <- sel_HctcuAxh:
			c.Post(false, sel_HctcuAxh)
			println("c")
		default:
			goChan.PostDefault()
			println("default")
		}
	}
}

func main() {
	goChan.Init()
	defer goChan.PrintTrace()

	test()

	time.Sleep(4 * time.Second)
}
```
We can now run the translated project and get a trace. One possible trace is
```
[signal(1, 2), signal(2, 3), signal(3, 4), signal(4, 5), pre(23, 3?, 4?, 5!, default), post(24, default)]
[wait(8, 2), pre(9, 2!), post(19, 2, 2!)]
[wait(10, 3), lock(11, 2, -, 1), pre(22, 2?)]
[wait(12, 4), lock(13, 1, -, 1), pre(14, 3!), post(15, 4, 3!), pre(16, 2?), post(17, 2, 2?, 9), unlock(18, 1)]
[wait(5, 5), lock(6, 2, r, 1), pre(7, 3?), post(20, 4, 3?, 14), unlock(21, 2)]
```
An explenation of the trace can be found in [goChan](https://github.com/ErikKassubek/GoChan/tree/main/goChan).
