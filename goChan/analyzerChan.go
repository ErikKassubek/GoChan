package goChan

import (
	"fmt"
	"math"
	"sort"
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
*/
type vcn struct {
	id       uint32
	routine  int
	position string
	send     bool
	pre      []int
	post     []int
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
Check for dangling events (events with pro but without post)
@return bool, true if dangling events exist, false otherwise
@return []uint32: list of chan with dangling events
*/
func checkForDanglingEvents() (bool, []uint32) {
	res := false
	resChan := make([]uint32, 0)
	for _, trace := range traces {
		for i, elem := range trace {
			switch pre := elem.(type) {
			case *TracePre:
				b := false
				for j := i + 1; j < len(trace) && j <= i+1; j++ {
					switch post := trace[j].(type) {
					case *TracePost:
						if pre.chanId == post.chanId && pre.send == post.send {
							b = true
						}
					}
					if b {
						break
					}
				}
				if !b {
					res = true
					resChan = append(resChan, pre.chanId)
				}
			case *TracePreSelect:
				b := false
				if pre.def { // no dangeling possible, if a default exist
					break
				}
				for j := i + 1; j < len(trace) && j <= i+1; j++ {
					switch post := trace[j].(type) {
					case *TracePost:
						if containsChan(post, pre.chanIds) {
							b = true
						}
					}
					if b {
						break
					}
				}
				if !b {
					res = true
					for _, preChan := range pre.chanIds {
						resChan = append(resChan, preChan.id)
					}
				}
			}
		}
	}
	return res, resChan
}

/*
Function to build a vector clock for a trace
@param c []uint32: List of chan ids with dangling post
@return []vcn: List of send and receive with pre and post vector clock annotation
*/
func buildVectorClockChan(c []uint32) []vcn {
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

		// fmt.Print("(", elem.routine + 1, ", ")
		// elem.elem.PrintElement()
		// fmt.Println(")")

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
							vectorClocks[int(t.GetTimestamp())][traceTotal[j].routine])
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
							vectorClocks[int(e.senderTimestamp)+1][traceTotal[j].routine])
						break
					}

				}
			}
		default:
			vectorClocks[i+1] = vectorClocks[i]
		}

		// fmt.Println(vectorClocks[i+1])
	}
	// build vector clock anotated traces
	vcTrace := make([]vcn, 0)

	for i, trace := range traces {
		for j, elem := range trace {
			switch pre := elem.(type) {
			case *TracePre: // normal pre
				if contains(c, pre.chanId) {
					b := false
					for k := j + 1; k < len(trace); k++ {
						switch post := trace[k].(type) {
						case *TracePost:
							if post.chanId == pre.chanId {
								vcTrace = append(vcTrace, vcn{id: pre.chanId, routine: i, position: pre.position, send: pre.send,
									pre: vectorClocks[int(pre.GetTimestamp())][i], post: vectorClocks[int(post.GetTimestamp())][i]})
								b = true
							}
						}
						if b {
							break
						}
					}
					if !b { // dangling event (pre without post)
						post_default_clock := make([]int, len(traces))
						for i := 0; i < len(traces); i++ {
							post_default_clock[i] = math.MaxInt
						}
						vcTrace = append(vcTrace, vcn{id: pre.chanId, routine: i, position: pre.position, send: pre.send,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock})
					}
				}
			case *TracePreSelect: // pre of select:
				channels := compaire(c, pre.chanIds)
				b1 := false
				for _, channel := range channels {
					b2 := false
					for k := j + 1; k < len(trace); k++ {
						switch post := trace[k].(type) {
						case *TracePost:
							if post.chanId == channel.id {
								vcTrace = append(vcTrace, vcn{id: channel.id, routine: i, position: pre.position, send: !channel.receive,
									pre: vectorClocks[int(pre.GetTimestamp())][i], post: vectorClocks[int(post.GetTimestamp())][i]})
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
					for _, channel := range channels {
						post_default_clock := make([]int, len(traces))
						for i := 0; i < len(traces); i++ {
							post_default_clock[i] = math.MaxInt
						}
						vcTrace = append(vcTrace, vcn{id: channel.id, routine: i, position: pre.position, send: !channel.receive,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock})
					}
				}
			}
		}
	}

	return vcTrace
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
			if elem1.id != elem2.id {
				continue
			}
			if vcUnComparable(elem1.pre, elem2.pre) || vcUnComparable(elem1.post, elem2.post) {
				if elem1.send && !elem2.send {
					collection[elem1.position] = append(collection[elem1.position], elem2.position)
				} else if elem2.send && !elem1.send {
					collection[elem2.position] = append(collection[elem2.position], elem1.position)
				}
			}
		}
	}
	res_string := make([]string, 0)
	for send, recs := range collection {
		res := fmt.Sprintf("Alternative Communication Partners:\n  %s", send)
		for _, rec := range recs {
			res += fmt.Sprintf("\n  -> %s", rec)
		}
		res_string = append(res_string, res)
	}
	return res_string
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
