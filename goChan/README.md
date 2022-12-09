# GoChan-Tracer

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## What?
GoChan-Tracer implements drop in replacements for channel operations in Go.
Those replacements can be used to create a trace of the executed program similar to the trace described in [1]. 

## Installation
```
go get github.com/ErikKassubek/GoChan/tracer
```
## How?
For the tracer, channel functionality is replaced by costum functions to create a trace of the program. The [instrumenter](https://github.com/ErikKassubek/GoChan/tree/main/instrumenter) can be used to automatically translate Go-Code into code with the replacements. 

## Example
Let's look at the following code:
```
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
By the [instrumenter](https://github.com/ErikKassubek/GoChan/tree/main/instrumenter) it gets translated into
```
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
	y := goChan.NewChan[int](0)

	a := goChan.NewChan[int](1)
	b := goChan.NewChan[int](0)
	c := goChan.NewChan[string](0)

	var l = goChan.NewMutex()
	var m = goChan.NewRWMutex()

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
	goChan.Init(10)
	defer goChan.RunAnalyzer()
	defer time.Sleep(time.Millisecond)
	test()
	time.Sleep(4 * time.Second)
}
``` 
One possible trace of the program is
```
[5: 0, 1: 0, 2: 0, 3: 1, 4: 0]
[
[signal(1, 2), signal(2, 3), signal(3, 4), signal(4, 5), pre(23, 3?, 4?, 5!, default), post(24, default)]
[wait(16, 2), pre(17, 2!), post(19, 2, 2!)]
[wait(8, 3), lock(18, 2, -, 1), pre(22, 2?)]
[wait(9, 4), lock(10, 1, -, 1), pre(11, 3!), post(12, 4, 3!), pre(13, 2?), post(20, 2, 2?, 17), unlock(21, 1)]
[wait(5, 5), lock(6, 2, r, 1), pre(7, 3?), post(14, 4, 3?, 11), unlock(15, 2)]
]
```
Every line represents a routine (the first line is the main routine).
The elements have the following meaning:
|Element|Meaning|
|-|-|
|signal(t, i) | a new routine with id = i has been created from the current routine|
|wait(t, i)| the current, non main routine was started with id = i|
|pre(t, i!)| the routine has reached a state, where channel i is supposed to send, but has not send yet|
|post(t, i, k!) | the channel k has successfully send its data in routine i with time step j|
|pre(t, i?)|the routine has reached a state, where channel i is supposed to receive, but has not received yet|
|post(t, i, k?, j)|the channel k has successfully received its data from routine i with time step j of routine i|
|pre(t, i?, j?, k?)| the routine has reached a select statements with cases for channels i, j and k. The select statement does not contain a default case. The statement has not yet executed.|
|pre(t, i?, j?, k?, default)| the routine has reached a select statements with cases for channels i, j and k. The select statement does contain a default case. The statement has not yet executed.|
|post(t, default)|The switch statement has executed and chosen the default case.|
|lock(t, i, j, l)|The lock with id i has tried to lock. l is 1 if the lock was successful, 0 if it was not (e.g. with TryLock). j can be "t", "r", "tr" or "-". "t" shows, that the lock operation was a try-lock operation. "r" shows, that it was an r-lock operation. "tr" shows, that it was a try-r-operation".|
|unlock(t, i)|The lock with id i was unlocked.|
t always states the timestamp of the operation.


## References 
[1] [M. Sulzmann and K. Stadtmüller, “Two-phase dynamic analysis of message-passing
go programs based on vector clocks,” CoRR, vol. abs/1807.03585, 2018.](https://arxiv.org/abs/1807.03585)
