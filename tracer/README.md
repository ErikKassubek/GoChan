# GoChan-Tracer

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## What?
GoChan-Tracer implements drop in replacements for channel operations in Go.
Those replacements can be used to create a trace of the executed program as described in [1]. 

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
	"time"
)

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
		y <- i
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

	time.Sleep(5 * time.Second)
}

```
By the [instrumenter](https://github.com/ErikKassubek/GoChan/tree/main/instrumenter) it gets translated into
```
package main

import (
	"github.com/ErikKassubek/GoChan/tracer"
	"time"
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
				y.Send(i)
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
		tracer.PreSelect(true, a.GetId(), b.GetId(), c.GetId())

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

	time.Sleep(5 * time.Second)
	tracer.PrintTrace()
}
``` 
By running this program we get the resulting trace as
```
[signal(2), signal(3), signal(4), signal(5), pre(3?, 4?, 5?, default), post(default)]
[wait(2), pre(1!), post(2, 1, 1!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?)]
[wait(5), pre(2?), post(4, 1, 2?)]
```
Every line represents a routine (the first line is the main routine).
The elements have the following meaning:
|Element|Meaning|
|-|-|
|signal(i) | a new routine with id = i has been created from the current routine|
|wait(i)| the current, non main routine was started with id = i|
|pre(i!)| the routine has reached a state, where channel i is supposed to send, but has not send yet|
|post(i, j, k?) | the channel k has successfully send its data in routine i with time step j|
|pre(i?)|the routine has reached a state, where channel i is supposed to receive, but has not received yet|
|post(i, j, k?)|the channel k has successfully received its data from routine i with time step j of routine i|
|pre(i?, j?, k?)| the routine has reached a select statements with cases for channels i, j and k. The select statement does not contain a default case. The statement has not yet executed.|
|pre(i?, j?, k?, default)| the routine has reached a select statements with cases for channels i, j and k. The select statement does contain a default case. The statement has not yet executed.|
|post(default)|The switch statement has executed and chosen the default case.|


A deeper explanation can be found in [1].

 
## References 
[1] [M. Sulzmann and K. Stadtmüller, “Two-phase dynamic analysis of message-passing
go programs based on vector clocks,” CoRR, vol. abs/1807.03585, 2018.](https://arxiv.org/abs/1807.03585)
