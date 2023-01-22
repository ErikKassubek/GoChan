# GoChan: Dynamic Analysis of Message Passing Go Programms

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## What
GoChan implements a dynamic detector for concurrency-bugs in Go.

The detector consists of an instrumenter and the detector-library.
The instrumenter transforms given code into code, which includes the GoChan 
detector. It will also create a new main file to run the instrumented code.

Written elaboration at: https://github.com/ErikKassubek/Bachelorarbeit

## How to use
To use the detector you need to clone or download this repository.
To instrument a program in \<input folder>, run 
```
make IN="<input folder>" EXEC="<executable name>"
```
\<executable name></executable> is the name of the executable produced by the code.
This will create a folder ./output. In this folder, there will be a folder 
containing the instrumented project. It will also contain a new main.go and 
a compiled main file.
The program can now by started by executing this main executable.
This will run the program and analyzer and at the end produce an output.

## Example
Assume we have a folder "project" containing:
```
./instrumenter (dir)
./Makefile
./program (dir)
```
where "intrumenter" is the folder containing the [instrumenter](https://github.com/ErikKassubek/GoChan/tree/main/instrumenter) and "Makefile" the [Makefile](https://github.com/ErikKassubek/GoChan/blob/main/Makefile).
The folder "program" is the folder containing the program witch is supposed to be analyzed. In this example it only contains the go.mod file 
```golang
module showProg

go 1.19
```
and one program file main.go
```golang
package main

import "sync"

func main() {
	var m sync.Mutex
	var n sync.Mutex

	c := make(chan int)

	go func() {
		close(c)
		<-c
	}()

	go func() {
		m.Lock()
		n.Lock()
		n.Unlock()
		m.Unlock()
		<-c
	}()

	n.Lock()
	m.Lock()
	m.Unlock()
	n.Unlock()

	c <- 1
}
```
In the "project" folder we can now run 
```shell
$ make IN="./show/" EXEC="showProg"
```
This will create an ./output folder in "project" containing 
```
program (dir)
main.go
main
```
The folder "main" contains the instrumented and compiled project. 
Depending on the project structure, "./output" can also contain other, empty
folders. They can be ignored. We can now run the "main" executable.
From this we get the following output
```
Determine switch execution order
Start Program Analysis
Analyse Program:   0%   
Analyse Program: 100%

Finish Analysis

Found Problems:

Potential Cyclic Mutex Locking:
Lock: /home/.../output/program/main.go:42
  Hs:
    /home/.../output/program/main.go:41
Lock: /home/.../output/program/main.go:33
  Hs:
    /home/.../output/program/main.go:32


Found dangling Events for Channels created at:  
    /home/.../output/program/main.go:16

  Possible Communication Partners:
    /home/.../output/program/main.go:45
    -> /home/.../output/program/main.go:23
    -> /home/.../output/program/main.go:36

Possible Send to Closed Channel:
    Close: /home/.../output/program/main.go:22
    Send: /home/.../output/program/main.go:45
```
In this example the paths are shortened for readability.

## Note
- The program must contain a go.mod file.
- Please be aware, that using external library functions which have Mutexe or 
channels as parameter or return values can lead to errors during the compilation.
- [GoImports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) must be installed
