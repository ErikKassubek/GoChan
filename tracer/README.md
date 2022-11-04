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
func main() {
	x := make(chan int, 0)
	y := make(chan string, 0)

	a := make(chan int, 0)
	b := make(chan int, 0)
	c := make(chan int, 0)
	d := make(chan int, 0)


	go func(c int, d string) { x <- i; a <- 1; fmt.Println(c, d) }(3, "a")
	go func() { <-x; x <- 1; b <- get_i(1) }()
	go func() { y <- "4"; <-x; c <- 5 }()
	go func() { i := <-y; d <- 6; fmt.Println(i) }()

	select {
	case <-a:
		fmt.Println("a")
	case <-b:
		fmt.Println("b")
	case <-c:
		fmt.Println("c")
	case <-d:
		fmt.Println("d")
	default:
		fmt.Println("default")
	}

	time.Sleep(2 * time.Second)
}
```
By the [instrumenter](https://github.com/ErikKassubek/GoChan/tree/main/instrumenter) it gets translated into
```
func main() {
	tracer.Init()

	x := tracer.NewChan[int](0)
	y := tracer.NewChan[string](0)

	a := tracer.NewChan[int](0)
	b := tracer.NewChan[int](0)
	c := tracer.NewChan[int](0)
	d := tracer.NewChan[int](0)

	tracer.Spawn(func(args_XoEFfRsWxP ...any) {
		var c int = args_XoEFfRsWxP[0].(int)
		var d string = args_XoEFfRsWxP[1].(string)
		x.Send(i)
		a.Send(1)
		fmt.fmt.Println(c, d)
	}, 3, "a")
	tracer.Spawn(func(args_LDnJObCsNV ...any) { x.Receive(); x.Send(1); b.Send(get_i(1)) })
	tracer.Spawn(func(args_lgTeMaPEZQ ...any) { y.Send("4"); x.Receive(); c.Send(5) })
	tracer.Spawn(func(args_leQYhYzRyW ...any) { i := y.Receive(); d.Send(6); fmt.Println(i) })

	{
		tracer.PreSelect(true, a.GetId(), b.GetId(), c.GetId(), d.GetId())

		select {
		case <-a.GetChan():
			fmt.Println("a")
		case <-b.GetChan():
			fmt.Println("b")
		case <-c.GetChan():
			fmt.Println("c")
		case <-d.GetChan():
			fmt.Println("d")
		default:
			fmt.Println("default")
			tracer.PostDefault()
		}
	}

	time.Sleep(2 * time.Second)
	tracer.PrintTrace()
}
``` 
By running this program we get the resulting trace as
```
[signal(2), signal(3), signal(4), signal(5), pre(3?, 4?, 5?, 6?, default), post(default)]
[wait(2), pre(1!), post(2, 1, 1!), pre(3!)]
[wait(3), pre(1?), post(2, 1, 1?), pre(1!), post(3, 2, 1!), pre(4!)]
[wait(4), pre(2!), post(4, 1, 2!), pre(1?), post(3, 2, 1?), pre(5!)]
[wait(5), pre(2?), post(4, 1, 2?), pre(6!)]
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
