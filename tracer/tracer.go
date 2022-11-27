package tracer

/*
Copyright (c) 2022, Erik Kassubek
All rights reserved.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

/*
Author: Erik Kassubek <erik-kassubek@t-online.de>
Package: GoChan-Tracer
Project: Bachelor Thesis at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Analysis of message passing go programs
*/

/*
trace.go
Drop in replacements of common channel functionality to create a trace of the
Program
*/

import (
	"sync"
	"time"

	"github.com/petermattis/goid"
)

var routineIndexLock sync.Mutex
var routineIndex = make(map[int64]int)
var numberRoutines = 0

var traces = make([]([]TraceElement), 0) // lists of traces
var tracesLock sync.RWMutex

var counter = make([]int, 0) // PC
var counterLock sync.RWMutex

func Init() {
	tracesLock.Lock()
	traces = append(traces, []TraceElement{})
	tracesLock.Unlock()

	counterLock.Lock()
	counter = append(counter, 0)
	counterLock.Unlock()

	routineIndexLock.Lock()
	routineIndex[goid.Get()] = 0
	routineIndexLock.Unlock()

	go func() { t := time.NewTimer(10 * time.Second); <-t.C; PrintTrace() }()
}

func PrintTrace() {
	tracesLock.RLock()
	for _, trace := range traces {
		print("[")
		for i, element := range trace {
			element.PrintElement()
			if i != len(trace)-1 {
				print(", ")
			}
		}
		println("]")
	}
	tracesLock.RUnlock()
}

// get the routine trace index
func getIndex() int {
	id := goid.Get()
	routineIndexLock.Lock()
	res := routineIndex[id]
	routineIndexLock.Unlock()
	return res
}

// increase the counter at a given index
func increaseCounter(index int) {
	counterLock.Lock()
	counter[index]++
	counterLock.Unlock()
}
