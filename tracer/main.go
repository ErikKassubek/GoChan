package main

import (
	"fmt"
	"time"
)

/*
func main() {

	x := make(chan int, 0)
	y := make(chan string, 0)

	a := make(chan int, 0)
	b := make(chan int, 0)
	c := make(chan int, 0)
	d := make(chan int, 0)

	go func() { x <- 1; a <- 0 }()
	go func() { <-x; x <- 2; b <- 0 }()
	go func() { y <- "a"; <-x; c <- 0 }()
	go func() { <-y; d <- 0 }()

	select {
	case <-a:
		println("a")
	case <-b:
		println("b")
	case <-c:
		println("c")
	case <-d:
		println("d")
	}

	time.Sleep(2 * time.Second)
}
*/

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

	tracer.PreSelect(a.GetId, b.GetId, c.GetId, d.GetId)
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
		print("def")
		tracer.PostDefault()
	}

	time.Sleep(2 * time.Second)
}
*/

/*
	func run(a int, b string) {
		fmt.Println(a, b)
	}
*/

/*
func run(args ...any) {
	a := args[0].(int)
	b := args[1].(string)
	fmt.Println(a, b)
}

func main() {
	Init()

	x := NewChan[int](0)
	y := NewChan[int](0)

	a := NewChan[int](0)
	b := NewChan[int](0)
	c := NewChan[int](0)
	d := NewChan[int](0)

	Spawn(run, 1, "a")
	Spawn(func(args ...any) { x.Send(1); fmt.Println(args[0].(string)) }, "q")
	Spawn(func(args ...any) { x.Receive(); x.Send(1) })
	Spawn(func(args ...any) { y.Send(1); x.Receive() })
	Spawn(func(args ...any) { y.Receive() })

	time.Sleep(2 * time.Second)

	PreSelect(true, a.GetId(), b.GetId(), c.GetId(), d.GetId())
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
		fmt.Println("def")
		PostDefault()
	}

	time.Sleep(1 * time.Second)
	PrintTrace()
}
*/

func main() {
	Init()

	x := NewChan[int](0)
	y := NewChan[int](0)

	a := NewChan[int](0)
	b := NewChan[int](0)
	c := NewChan[int](0)
	d := NewChan[int](0)
	e := NewChan[string](0)
	Spawn(func(args ...any) { x.Send(1); fmt.Println(args[0].(string)) }, "q")
	Spawn(func(args ...any) { x.Receive(); x.Send(1) })
	Spawn(func(args ...any) { y.Send(1); x.Receive() })
	Spawn(func(args ...any) { y.Receive() })
	{
		PreSelect(true, a.GetId(), b.GetId(), c.GetId(), d.GetId())

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
			PostDefault()
		}
	}

	e.Close()

	time.Sleep(2 * time.Second)

	PrintTrace()
}
