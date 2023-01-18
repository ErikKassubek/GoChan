package goChan

import (
	"fmt"
)

/*
Copyright (c) 2023, Erik Kassubek
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
Package: goChan
Project: Bachelor Thesis at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Analysis of message passing go programs
*/

/*
tracerElements.go
Type declarations for the trace elements
*/

/*
Interface for a trace element.
@signature PrintElement(): function to print the element
*/
type TraceElement interface {
	GetTimestamp() uint32
	PrintElement()
}

// ==================== Channel =====================

/*
Struct for a signal in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field routine uint32: id of the new routine
*/
type TraceSignal struct {
	position  string
	timestamp uint32
	routine   uint32
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (ts *TraceSignal) GetTimestamp() uint32 {
	return ts.timestamp
}

/*
Function to print the signal trace element
@receiver *TraceSignal
*/
func (ts *TraceSignal) PrintElement() {
	fmt.Printf("signal(%d, %d)", ts.timestamp, ts.routine+1)
}

/*
Struct for a wait in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field routine uint32: id of the routine
*/
type TraceWait struct {
	position  string
	timestamp uint32
	routine   uint32
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (ts *TraceWait) GetTimestamp() uint32 {
	return ts.timestamp
}

/*
Function to print the wait trace element
@receiver *TraceWait
*/
func (tw *TraceWait) PrintElement() {
	fmt.Printf("wait(%d, %d)", tw.timestamp, tw.routine+1)
}

/*
Struct for a pre in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field chanId uint32: id of the Chan
@field chanCreation string: pos of the creation of the chan
@field send bool: true if it is a preSend, false otherwise
*/
type TracePre struct {
	position     string
	timestamp    uint32
	chanId       uint32
	chanCreation string
	send         bool
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tp *TracePre) GetTimestamp() uint32 {
	return tp.timestamp
}

/*
Function to print the pre trace element
@receiver *TracePre
*/
func (tp *TracePre) PrintElement() {
	direction := "?"
	if tp.send {
		direction = "!"
	}
	fmt.Printf("pre(%d, %d%s)", tp.timestamp, tp.chanId, direction)
}

/*
Struct for a post in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field chanId uint32: id of the Chan
@field chanCreation string: pos of the creation of the chan
@field send bool: true if it is a preSend, false otherwise
@field senderId: id of the sender of the message
@field senderTimestamp: timestamp of the sender at send
@field noComs: if send, number of send on the channel, if read number of read (after the post)
*/
type TracePost struct {
	position        string
	timestamp       uint32
	chanId          uint32
	chanCreation    string
	send            bool
	senderId        uint32
	senderTimestamp uint32
	noComs          int
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tp *TracePost) GetTimestamp() uint32 {
	return tp.timestamp
}

/*
Function to print the post trace element
@receiver *TracePost
*/
func (tp *TracePost) PrintElement() {
	if tp.send {
		direction := "!"
		fmt.Printf("post(%d, %d, %d%s, %d)", tp.timestamp, tp.senderId, tp.chanId, direction, tp.noComs)
	} else {
		direction := "?"
		fmt.Printf("post(%d, %d, %d%s, %d, %d)", tp.timestamp, tp.senderId, tp.chanId, direction, tp.senderTimestamp, tp.noComs)
	}
}

/*
Struct for a close in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field chanId uint32: id of the Chan
@field chanCreation string: pos of the creation of the chan
*/
type TraceClose struct {
	position     string
	timestamp    uint32
	chanId       uint32
	chanCreation string
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tc *TraceClose) GetTimestamp() uint32 {
	return tc.timestamp
}

/*
Function to print the close trace element
@receiver *TraceClose
*/
func (tc *TraceClose) PrintElement() {
	fmt.Printf("close(%d, %d)", tc.timestamp, tc.chanId)
}

/*
Struct for a preSelect in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field chanIds []PreObj: list of channels in cases
@field def bool: true if the select has a default case, false otherwise
*/
type TracePreSelect struct {
	position  string
	timestamp uint32
	chanIds   []PreObj
	def       bool // true if select has default case
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tps *TracePreSelect) GetTimestamp() uint32 {
	return tps.timestamp
}

/*
Function to print the preSelect trace element
@receiver *TracePreSelect
*/
func (tps *TracePreSelect) PrintElement() {
	fmt.Printf("pre(%d, ", tps.timestamp)
	for i, c := range tps.chanIds {
		if c.receive {
			fmt.Printf("%d?", c.id)
		} else {
			fmt.Printf("%d!", c.id)
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

/*
Struct for a default in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
*/
type TraceDefault struct {
	position  string
	timestamp uint32
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (td *TraceDefault) GetTimestamp() uint32 {
	return td.timestamp
}

/*
Function to print the default trace element
@receiver *TraceDefault
*/
func (td *TraceDefault) PrintElement() {
	fmt.Printf("post(%d, default)", td.timestamp)
}

// ==================== Mutex =====================

/*
Struct for a lock in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field lockId uint32: id of the Mutex
@field mutexCreation string: position of the creation of the mutex
@field try bool: true if it is a try-lock, false otherwise
@field read bool: true if it is a r-lock, false otherwise
@field suc bool: true if the operation was successful, false otherwise (only try)
*/
type TraceLock struct {
	position      string
	timestamp     uint32
	lockId        uint32
	mutexCreation string
	try           bool
	read          bool
	suc           bool
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tl *TraceLock) GetTimestamp() uint32 {
	return tl.timestamp
}

/*
Function to print the lock trace element
@receiver *TraceLock
*/
func (tl *TraceLock) PrintElement() {
	p_elem := ""
	if tl.try {
		p_elem += "t"
	}
	if tl.read {
		p_elem += "r"
	}
	if p_elem == "" {
		p_elem = "-"
	}
	var suc_elem string
	if tl.suc {
		suc_elem += "1"
	} else {
		suc_elem += "0"
	}
	fmt.Printf("lock(%d, %d, %s, %s)", tl.timestamp, tl.lockId, p_elem, suc_elem)
}

/*
Struct for a unlock in the trace.
@field postion string: string describing the position of the creation in code
@field timestamp uint32: timestamp of the creation of the trace object
@field lockId uint32: id of the Mutex
@field mutexCreation string: position of the creation of the mutex
*/
type TraceUnlock struct {
	position      string
	timestamp     uint32
	lockId        uint32
	mutexCreation string
}

/*
Function to return the timestamp
@return uint32: timestamp
*/
func (tu *TraceUnlock) GetTimestamp() uint32 {
	return tu.timestamp
}

/*
Function to print the unlock trace element
@receiver *TraceUnlock
*/
func (tu *TraceUnlock) PrintElement() {
	fmt.Printf("unlock(%d, %d)", tu.timestamp, tu.lockId)
}
