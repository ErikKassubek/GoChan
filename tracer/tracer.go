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
var routineIndex = make(map[int64]uint32)

var numberRoutines uint32
var numberOfChan uint32
var numberOfMutex uint32

var traces = make([]([]TraceElement), 0) // lists of traces
var tracesLock sync.RWMutex

var counter uint32 // PC

/*
Function to initialize the tracer.
@return: nil
*/
func Init() {
	numberRoutines = 0
	numberOfChan = 0
	numberOfMutex = 0
	counter = 0

	tracesLock.Lock()
	traces = append(traces, []TraceElement{})
	tracesLock.Unlock()

	routineIndexLock.Lock()
	routineIndex[goid.Get()] = 0
	routineIndexLock.Unlock()

	go func() { t := time.NewTimer(10 * time.Second); <-t.C; PrintTrace() }()
}

/*
Function to print the collected trace.
@return nil
*/
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

/*
Function to get the index of the routine, from wich the function is called
@return nil
*/
func getIndex() uint32 {
	id := goid.Get()
	routineIndexLock.Lock()
	res := routineIndex[id]
	routineIndexLock.Unlock()
	return res
}
