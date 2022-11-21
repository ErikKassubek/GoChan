package tracer

import (
	"fmt"
)

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
traceElements.go
Type declarations for the trace elements
*/

// interface for a trace element
type TraceElement interface {
	PrintElement()
}

// ==================== Channel =====================

// type for the signal element
type TraceSignal struct {
	routine int
}

// print function for TraceSignal
func (ts *TraceSignal) PrintElement() {
	fmt.Printf("signal(%d)", ts.routine+1)
}

// type for the wait element
type TraceWait struct {
	routine int
}

// print function for TraceWait
func (tw *TraceWait) PrintElement() {
	fmt.Printf("wait(%d)", tw.routine+1)
}

// type for the pre element
type TracePre struct {
	chanId int
	send   bool
}

// print function for TracePre
func (tp *TracePre) PrintElement() {
	direction := "?"
	if tp.send {
		direction = "!"
	}
	fmt.Printf("pre(%d%s)", tp.chanId+1, direction)
}

// type for the post element
type TracePost struct {
	chanId    int
	send      bool
	SenderId  int
	timestamp int
}

// print function for TracePost
func (tp *TracePost) PrintElement() {
	direction := "?"
	if tp.send {
		direction = "!"
	}
	fmt.Printf("post(%d, %d, %d%s)", tp.SenderId+1, tp.timestamp, tp.chanId+1, direction)
}

// type for the close element
type TraceClose struct {
	chanId int
}

// print function for TraceClose
func (tc *TraceClose) PrintElement() {
	fmt.Printf("close(%d)", tc.chanId+1)
}

// type for pre select event
type TracePreSelect struct {
	chanIds []PreObj
	def     bool // true if select has default case
}

func (tps *TracePreSelect) PrintElement() {
	fmt.Printf("pre(")
	for i, c := range tps.chanIds {
		if c.receive {
			fmt.Printf("%d?", c.id+1)
		} else {
			fmt.Printf("%d!", c.id+1)
		}
		if i != len(tps.chanIds)-1 {
			fmt.Printf(", ")
		}
	}
	if tps.def {
		fmt.Printf(", default")
	}
	fmt.Printf(")")
}

// type for the default element
type TraceDefault struct{}

// print function for TracePre
func (td *TraceDefault) PrintElement() {
	fmt.Printf("post(default)")
}

// ==================== Mutex =====================

type TraceLock struct {
	lockId int
	try    bool
	read   bool
	suc    bool
}

func (tl *TraceLock) PrintElement() {
	p_elem := ""
	if tl.try {
		p_elem += "t, "
	}
	if tl.read {
		p_elem += "r, "
	}
	if tl.suc {
		p_elem += "1"
	} else {
		p_elem += "0"
	}
	fmt.Printf("lock(%d, %s)", tl.lockId+1, p_elem)
}

type TraceUnlock struct {
	lockId int
}

func (tu *TraceUnlock) PrintElement() {
	fmt.Printf("unlock(%d)", tu.lockId+1)
}
