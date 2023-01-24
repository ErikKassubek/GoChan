package goChan

import (
	"fmt"
	"math"
	"sort"
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
analyzerMutex.go
Analyze the trace to check for deadlocks containing only (rw-)mutexe based on
the UNDEAD algorithm.
*/

/*
Struct to save the pre and post vector clocks of a channel operation
@field id uint32: id of the channel
@field send bool: true if it is a send event, false otherwise
@field pre []uint32: pre vector clock
@field post []uint32: post vector clock
@field noComs: if send number of completed sends on the channel, otherwise number of completed receives
*/
type vcn struct {
	id       uint32
	creation string
	routine  int
	position string
	send     bool
	pre      []int
	post     []int
	noComs   int
}

/*
Struct to save an element of the complete type
@field routine int: number of routine
@field elem TraceElement: element of the trace
*/
type tte struct {
	routine uint32
	elem    TraceElement
}

/*
Functions to implement the sort.Interface
*/
type ttes []tte

func (s ttes) Len() int {
	return len(s)
}
func (s ttes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ttes) Less(i, j int) bool {
	return s[i].elem.GetTimestamp() < s[j].elem.GetTimestamp()
}

/*
Function to build a vector clock for a trace and search for dangling events
@return []vcn: List of send and receive with pre and post vector clock annotation
@return bool, true if dangling events exist, false otherwise
@return []string: list of creation positions of chans with dangling events
*/
func buildVectorClockChan() ([]vcn, bool, []string) {
	// build one trace with all elements in the form [(routine, elem), ...]
	var traceTotal ttes

	for i, trace := range traces {
		for _, elem := range trace {
			traceTotal = append(traceTotal, tte{uint32(i), elem})
		}
	}

	sort.Sort(traceTotal)

	// map the timestep to the vector clock
	vectorClocks := make(map[int][][]int)
	vectorClocks[0] = make([][]int, len(traces))
	for i := 0; i < len(traces); i++ {
		vectorClocks[0][i] = make([]int, len(traces))
	}

	for i, elem := range traceTotal {

		// int("(", elem.routine + 1, ", ")
		// elem.elem.PrintElement()
		// intln(")")

		switch e := elem.elem.(type) {
		case *TraceSignal:
			vectorClocks[i+1] = update_send(vectorClocks[i], int(elem.routine))
		case *TraceWait:
			b := false
			for j := i - 1; j >= 0; j-- {
				switch t := traceTotal[j].elem.(type) {
				case *TraceSignal:
					if t.routine == e.routine {
						vectorClocks[i+1] = update_reveive(vectorClocks[i], int(e.routine), int(traceTotal[j].routine),
							vectorClocks[int(t.GetTimestamp())-1][traceTotal[j].routine])
						b = true
					}
				}
				if b {
					break
				}
			}
		case *TracePost:
			if e.send {
				vectorClocks[i+1] = update_send(vectorClocks[i], int(elem.routine))
			} else {
				for j := i - 1; j >= 0; j-- {
					if e.senderTimestamp == traceTotal[j].elem.GetTimestamp() {
						vectorClocks[i+1] = update_reveive(vectorClocks[i], int(elem.routine), int(traceTotal[j].routine),
							vectorClocks[int(e.senderTimestamp)-1][traceTotal[j].routine])
						break
					}

				}
			}
		default:
			vectorClocks[i+1] = vectorClocks[i]
		}

		// intln(vectorClocks[i+1])
	}
	// build vector clock anotated traces
	vcTrace := make([]vcn, 0)

	ok := false
	danglingEvents := make([]string, 0)

	for i, trace := range traces {
		for j, elem := range trace {
			switch pre := elem.(type) {
			case *TracePre: // normal pre
				b := false
				for k := j + 1; k < len(trace); k++ {
					switch post := trace[k].(type) {
					case *TracePost:
						if post.chanId == pre.chanId &&
							len(vectorClocks[int(pre.GetTimestamp())]) > i &&
							len(vectorClocks[int(pre.GetTimestamp())]) > i {
							vcTrace = append(vcTrace, vcn{id: pre.chanId, creation: pre.chanCreation, routine: i, position: pre.position, send: pre.send,
								pre: vectorClocks[int(pre.GetTimestamp())][i], post: vectorClocks[int(post.GetTimestamp())][i],
								noComs: post.noComs})
							b = true
						}
					}
					if b {
						break
					}
				}
				if !b { // dangling event (pre without post)
					ok = true
					danglingEvents = append(danglingEvents, pre.chanCreation)
					post_default_clock := make([]int, len(traces))
					for i := 0; i < len(traces); i++ {
						post_default_clock[i] = math.MaxInt
					}
					fmt.Println(vectorClocks[int(pre.GetTimestamp())])
					if len(vectorClocks[int(pre.GetTimestamp())]) > i {
						vcTrace = append(vcTrace, vcn{id: pre.chanId, creation: pre.chanCreation, routine: i, position: pre.position, send: pre.send,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock, noComs: -1})
					}

				}
			case *TracePreSelect: // pre of select:
				b1 := false
				for _, channel := range pre.chanIds {
					b2 := false
					for k := j + 1; k < len(trace); k++ {
						switch post := trace[k].(type) {
						case *TracePost:
							if post.chanId == channel.id {
								vcTrace = append(vcTrace, vcn{id: channel.id, creation: post.chanCreation, routine: i, position: pre.position, send: !channel.receive,
									pre: vectorClocks[int(pre.GetTimestamp())][i], post: vectorClocks[int(post.GetTimestamp())][i],
									noComs: post.noComs})
								b1 = true
								b2 = true
							}
						}
						if b2 {
							break
						}
					}
					if b1 {
						break
					}
				}
				if !b1 { // dangling event
					for _, channel := range pre.chanIds {
						ok = true
						danglingEvents = append(danglingEvents, pre.position)
						post_default_clock := make([]int, len(traces))
						for i := 0; i < len(traces); i++ {
							post_default_clock[i] = math.MaxInt
						}
						vcTrace = append(vcTrace, vcn{id: channel.id, creation: channel.chanCreation, routine: i, position: pre.position, send: !channel.receive,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock, noComs: -1})
					}
				}
			case *TraceClose:
				vcTrace = append(vcTrace, vcn{id: pre.chanId, creation: pre.chanCreation, routine: i, position: pre.position, pre: vectorClocks[int(pre.GetTimestamp())][i],
					post: vectorClocks[int(pre.GetTimestamp())][i], noComs: -1})
			}
		}
	}

	return vcTrace, ok, danglingEvents
}

/*
Find alternative communications based on vector clock annotated events
@param vcTrace []vcn: vector clock annotated events
@return []string: list of found communications
*/
func findAlternativeCommunication(vcTrace []vcn) []string {
	collection := make(map[string][]string)
	for i := 0; i < len(vcTrace)-1; i++ {
		for j := i + 1; j < len(vcTrace); j++ {
			elem1 := vcTrace[i]
			elem2 := vcTrace[j]
			if !isComm(elem1) || !isComm(elem2) { // one of the elements is close
				continue
			}
			if elem1.id != elem2.id { // must be same channel
				continue
			}
			if elem1.send == elem2.send { // must be send and receive
				continue
			}
			if !elem1.send { // swap elems sucht that 1 is send and 2 is receive
				elem1, elem2 = elem2, elem1
			}
			// add empty list of send if nessessary
			if len(collection[elem1.position]) == 0 {
				collection[elem1.position] = make([]string, 0)
			}
			if (vcUnComparable(elem1.pre, elem2.pre) || vcUnComparable(elem1.post, elem2.post) ||
				vcUnComparable(elem1.pre, elem2.post) || vcUnComparable(elem1.post, elem2.pre)) ||
				(getChanSize(elem1.id) != 0 &&
					distance(elem1.noComs, elem2.noComs) < chanSize[elem1.id]) {
				collection[elem1.position] = append(collection[elem1.position], elem2.position)
			}
		}
	}
	res_string := make([]string, 0)
	for send, recs := range collection {
		res := fmt.Sprintf("  Possible Communication Partners:\n    %s", send)
		in := make(map[string]int)
		if len(recs) == 0 {
			res += "\n    -> No possible communication found"
		}
		for _, rec := range recs {
			if _, ok := in[rec]; !ok {
				res += fmt.Sprintf("\n    -> %s", rec)
				in[rec] = 0
			}
		}
		res_string = append(res_string, res)
	}
	return res_string
}

/*
Function to find situation where a send to a closed channel is possible
@param vcTrace []vcn: list of vector-clock annotated events
@return bool: true if a possible send to close is found, false otherwise
@return []string: list of possible send to close
*/
func checkForPossibleSendToClosed(vcTrace []vcn) (bool, []string) {
	res := make([]string, 0)
	r := false
	// search for pre select
	for _, trace := range traces {
		for _, elem := range trace {
			switch sel := elem.(type) {
			case *TraceClose:
				// get vector clocks of pre select
				var preVc []int
				var postVc []int
				for _, clock := range vcTrace {
					if sel.position == clock.position {
						preVc = clock.pre
						postVc = clock.post
					}
				}

				// find possible pre vector clocks
				for _, vc := range vcTrace {
					if vc.id == sel.chanId && vc.send && (vcUnComparable(preVc, vc.pre) || vcUnComparable(postVc, vc.post)) {
						r = true
						res = append(res, fmt.Sprintf("Possible Send to Closed Channel:\n    Close: %s\n    Send: %s", sel.position, vc.position))
					}
				}
			}
		}
	}
	return r, res
}

/*
Check for buffered channels where a message was written to the channel but
never read
@param vcTrace []vcn: vector clock annotated trace
@return bool: true if an non empty chan was found, false otherwise
@return []string: messages
*/
func checkForNonEmptyChan(vcTrace []vcn) (bool, []string) {
	numberMessages := make(map[string]int)
	resString := make([]string, 0)
	res := false
	for _, message := range vcTrace {
		if !isComm(message) {
			continue
		}
		_, ok := numberMessages[message.creation]
		if !ok {
			numberMessages[message.creation] = 0
		}
		if message.send {
			if message.post[0] < math.MaxInt {
				numberMessages[message.creation]++
			}
		} else {
			if message.post[0] < math.MaxInt {
				numberMessages[message.creation]--
			}
		}
	}
	for position, value := range numberMessages {
		if value > 0 {
			resString = append(resString, fmt.Sprintf("%d unread message(s) in Channel created at %s", value, position))
			res = true
		}
	}
	return res, resString
}

/*
Test weather 2 vector clocks are incomparable
@param vc1 []int: first vector clock
@param vc2 []int: second vector clock
@return bool: true, if vc1 and vc2 are uncomparable, false otherwise
*/
func vcUnComparable(vc1, vc2 []int) bool {
	gr := false
	lt := false
	for i := 0; i < len(vc1); i++ {
		if vc1[i] > vc2[i] {
			gr = true
		} else if vc1[i] < vc2[i] {
			lt = true
		}

		if gr && lt {
			return true
		}
	}
	return false
}

/*
Function to create a new vector clock stack after a send event
@param vectorClock [][]int: old vector clock stack
@param i int: routine of sender
@return [][]int: new vector clock stack
*/
func update_send(vectorClock [][]int, i int) [][]int {
	c := make([][]int, len(vectorClock))
	for i := range vectorClock {
		c[i] = make([]int, len(vectorClock[i]))
		copy(c[i], vectorClock[i])
	}
	c[i][i]++
	return c
}

/*
Function to create a new vector clock stack after a receive statement
@param vectorClock [][]int: old vector clock stack
@param routineRec int: routine of receiver
@param routineSend int: routine of sender
@param vectorClockSender []int: vector clock of the sender at time of sending
@ret [][] int: new vector clock stack
*/
func update_reveive(vectorClock [][]int, routineRec int, routineSend int, vectorClockSender []int) [][]int {
	c := make([][]int, len(vectorClock))
	for i := range vectorClock {
		c[i] = make([]int, len(vectorClock[i]))
		copy(c[i], vectorClock[i])
	}

	c[routineRec][routineRec]++

	if c[routineRec][routineRec] <= vectorClockSender[routineRec] {
		c[routineRec][routineRec] = vectorClockSender[routineRec] + 1
	}

	for l := 0; l < len(c[routineRec]); l++ {
		if c[routineRec][l] < vectorClockSender[l] {
			c[routineRec][l] = vectorClockSender[l]
		}
	}

	return c
}

/*
Check if elem is in list
@param list []uint32: list
@param elem uint32: elem
@return bool: true if elem in list, false otherwise
*/
func contains(list []uint32, elem uint32) bool {
	for _, e := range list {
		if e == elem {
			return true
		}
	}
	return false
}

/*
Check if a TracePost corresponds to an element in an PreOpj list
created by an TracePreSelect
@param list []PreOpj: list of PreOps elements
@param elem TracePost: post event
@return bool: true, if elem corresponds to an element in list, false otherwise
*/
func containsChan(elem *TracePost, list []PreObj) bool {
	for _, pre := range list {
		if pre.id == elem.chanId && pre.receive != elem.send {
			return true
		}
	}
	return false
}

/*
Get a list of all cases in a pre select which are in listId
@param listId []uint32: list of ids
@param listPreObj []PreObj: list of PreObjs as created by a pre select
@return []PreObj: list of preObj from listPreObj, where the channel is in listId
*/
func compaire(listId []uint32, listPreObj []PreObj) []PreObj {
	res := make([]PreObj, 0)
	for _, id := range listId {
		for _, pre := range listPreObj {
			if id == pre.id {
				res = append(res, pre)
			}
		}
	}
	return res
}

/*
Get the capacity of a channel
@param index int: id of the channel
@return int: size of the channel
*/
func getChanSize(index uint32) int {
	chanSizeLock.Lock()
	size := chanSize[index]
	chanSizeLock.Unlock()
	return size
}

/*
Check wether a vcn describes a communication
@param v vcn: vcn to test
@return bool: true is communication, false if not (mainly close)
*/
func isComm(v vcn) bool {
	if len(v.pre) != len(v.post) {
		return false
	}
	for i := 0; i < len(v.pre); i++ {
		if v.pre[i] != v.post[i] {
			return true
		}
	}
	return false
}

/*
Calculate the absolute difference between x and y
@param x uint32
@param y uint32
@return int: |x-y|
*/
func distance(x int, y int) int {
	if x > y {
		return x - y
	} else {
		return y - x
	}
}
