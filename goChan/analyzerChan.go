package main

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
@field preTime uint32: timestamp of the pre event
@field send bool: true if it is a send event, false otherwise
@field pre []uint32: pre vector clock
@field post []uint32: post vector clock
@field first int: number of Send strongly before send or difference between number receive strongly before receive and receive independent of receive
@field second int: sum of number send strongly before send and send independent of send or sum of number receive strongly before receive and receive independent of receive
*/
type vcn struct {
	id       uint32
	preTime  uint32
	creation string
	routine  int
	position string
	send     bool
	pre      []int
	post     []int
	first    int
	second   int
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
Struct to save calculation values for buffered channels
@field first int
@field second int
*/
type calcVal struct {
	first  int
	second int
}

/*
Struct to save combination of pos info and pre-timestap
@field preTime uint32: timestamp of the pre event
@field pod sting: info
*/
type infoTime struct {
	preTime uint32
	pos     string
}

/*
Function to build a vector clock for a trace and search for dangling events
@return []vcn: List of send and receive with pre and post vector clock annotation
@return map[infoTime]int: number of sends/receive on same channel before or cuncurrent with send/receive
@return map[infoTime]int: number of sends/receive on same channel strictly before send/receive
*/
func buildVectorClockChan() ([]vcn, map[infoTime]int, map[infoTime]int) {
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
							vcTrace = append(vcTrace, vcn{id: pre.chanId, preTime: pre.timestamp,
								creation: pre.chanCreation, routine: i, position: pre.position, send: pre.send,
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
					if len(vectorClocks[int(pre.GetTimestamp())]) > i {
						vcTrace = append(vcTrace, vcn{id: pre.chanId, preTime: pre.timestamp,
							creation: pre.chanCreation, routine: i, position: pre.position, send: pre.send,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock})
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
								vcTrace = append(vcTrace, vcn{id: channel.id, preTime: pre.timestamp,
									creation: post.chanCreation, routine: i, position: pre.position, send: !channel.receive,
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
					for _, channel := range pre.chanIds {
						post_default_clock := make([]int, len(traces))
						for i := 0; i < len(traces); i++ {
							post_default_clock[i] = math.MaxInt
						}
						vcTrace = append(vcTrace, vcn{id: channel.id, preTime: pre.timestamp,
							creation: channel.chanCreation, routine: i, position: pre.position, send: !channel.receive,
							pre: vectorClocks[int(pre.GetTimestamp())][i], post: post_default_clock})
					}
				}
			case *TraceClose:
				vcTrace = append(vcTrace, vcn{id: pre.chanId, preTime: pre.timestamp,
					creation: pre.chanCreation, routine: i, position: pre.position, pre: vectorClocks[int(pre.GetTimestamp())][i],
					post: vectorClocks[int(pre.GetTimestamp())][i]})
			}
		}
	}

	// calculate first and second values as well as the number of
	// sends or receives beforeOrEqual an operation on a buffered channel
	beforeOrEqual := make(map[infoTime]int)
	before := make(map[infoTime]int)

	for i := 0; i < len(vcTrace); i++ {
		if getChanSize(vcTrace[i].id) == 0 {
			continue
		}
		sp := 0 // no of send before send in i
		su := 0 // no of send independent of send in i
		rp := 0 // no of receive before send in i
		ru := 0 // no of receive independent of receive in i
		for j := 0; j < len(vcTrace); j++ {
			if i == j {
				continue
			}
			if vcTrace[i].id != vcTrace[j].id {
				continue
			}
			indep := vcUnComparable(vcTrace[i].pre, vcTrace[j].pre)
			if vcTrace[j].send {
				if indep {
					su += 1
				} else if vcTrace[i].routine == vcTrace[j].routine && j < i {
					sp += 1
				}
			} else {
				if indep {
					ru += 1
				} else if vcTrace[i].routine == vcTrace[j].routine && j < i {
					rp += 1
				}
			}

			// check if send or receive on buffered before
			if i > j {
				continue
			}
			if vcTrace[i].send == vcTrace[j].send {
				if vcIsBeforeOrConcurrent(vcTrace[i].pre, vcTrace[j].pre) || vcIsBeforeOrConcurrent(vcTrace[i].post, vcTrace[j].post) {
					jIt := infoTime{preTime: vcTrace[j].preTime, pos: vcTrace[j].position}
					beforeOrEqual[jIt] = beforeOrEqual[jIt] + 1
					if vcIsBefore(vcTrace[i].pre, vcTrace[j].pre) || vcIsBefore(vcTrace[i].post, vcTrace[j].post) {
						before[jIt] = before[jIt] + 1
					}
				}
				if vcIsBeforeOrConcurrent(vcTrace[j].pre, vcTrace[i].pre) || vcIsBeforeOrConcurrent(vcTrace[j].post, vcTrace[i].post) {
					iIt := infoTime{preTime: vcTrace[i].preTime, pos: vcTrace[i].position}
					beforeOrEqual[iIt] = beforeOrEqual[iIt] + 1
					if vcIsBefore(vcTrace[j].pre, vcTrace[i].pre) || vcIsBefore(vcTrace[j].post, vcTrace[i].post) {
						before[iIt] = before[iIt] + 1
					}
				}
			}
		}
		if vcTrace[i].send {
			vcTrace[i].first = sp
			vcTrace[i].second = sp + su
		} else {
			vcTrace[i].first = rp - ru
			vcTrace[i].second = rp + ru
		}
	}
	return vcTrace, beforeOrEqual, before
}

/*
Find alternative communications based on vector clock annotated events
@param vcTrace []vcn: vector clock annotated events
@param before map[infoTime]int: number of send/receive strictly before send/receive on buffered channel
@return map[string][]string: map for possible communications from send to receive
@return []string: list of all sends
@return []string: list of all receives
*/
func findAlternativeCommunication(vcTrace []vcn, beforeCurrent map[infoTime]int, before map[infoTime]int) (map[infoTime][]infoTime, []infoTime, []infoTime) {
	collection := make(map[infoTime][]infoTime)
	listOfSends := make([]infoTime, 0)
	listOfReceive := make([]infoTime, 0)
	for i := 0; i < len(vcTrace); i++ {
		// collect send and receive
		if isComm(vcTrace[i]) {
			if vcTrace[i].send {
				listOfSends = append(listOfSends, infoTime{preTime: vcTrace[i].preTime, pos: vcTrace[i].position})
			} else {
				listOfReceive = append(listOfReceive, infoTime{preTime: vcTrace[i].preTime, pos: vcTrace[i].position})
			}
		}
		// find possible pairs
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
			it1 := infoTime{preTime: elem1.preTime, pos: elem1.position}
			it2 := infoTime{preTime: elem2.preTime, pos: elem2.position}
			// add empty list of send if nessessary
			if len(collection[it1]) == 0 {
				collection[it1] = make([]infoTime, 0)
			}
			uncomp1 := vcUnComparable(elem1.pre, elem2.pre)
			uncomp4 := vcUnComparable(elem1.post, elem2.post)
			if (getChanSize(elem1.id) == 0 && (uncomp1 || uncomp4)) ||
				(getChanSize(elem1.id) != 0 && beforeCurrent[it1] >= beforeCurrent[it2] && before[it1] <= before[it2]) {
				collection[it1] = append(collection[it1], infoTime{preTime: elem2.preTime, pos: elem2.position})
			}
		}
	}

	return collection, listOfSends, listOfReceive
}

/*
Function to rearrange communications and find invalid paths
@param rs map[string][]string: map of possible communications
@param vcTrace []vcn: vector annotated trace
@param listOfSends []string: list of all send
@param listOfReceive []string: list of all receive
*/
func findPossibleInvalidCommunications(rs map[infoTime][]infoTime, vcTrace []vcn,
	listOfSends []infoTime, listOfReceive []infoTime) string {
	mapOfReceive := make(map[infoTime][]infoTime)

	for _, vc := range vcTrace {
		if vc.send || !isComm(vc) {
			continue
		}
		pIt := infoTime{preTime: vc.preTime, pos: vc.position}
		if _, ok := mapOfReceive[pIt]; !ok {
			mapOfReceive[pIt] = make([]infoTime, 0)
		}
	}

	// get all send and receive
	for s, r := range rs {
		for _, res := range r {
			mapOfReceive[res] = append(mapOfReceive[res], s)
		}
	}

	resString := ""

	// get errors on send
	for i := 0; i < len(listOfSends); i++ {
		resString += findImpossibleCommunication(rs, listOfSends, make(map[infoTime]infoTime), true, vcTrace)
		if len(listOfSends) > 1 {
			first, newList := listOfSends[0], listOfSends[1:]
			newList = append(newList, first)
			listOfSends = newList
		}
	}

	// get errors on receive
	for i := 0; i < len(listOfReceive); i++ {
		resString += findImpossibleCommunication(mapOfReceive, listOfReceive, make(map[infoTime]infoTime), false, vcTrace)
		if len(listOfReceive) > 1 {
			first, newList := listOfReceive[0], listOfReceive[1:]
			newList = append(newList, first)
			listOfReceive = newList
		}
	}
	return resString
}

/*
Function to find runs which lead to problems
@param rs map[string][]string: possible communications
@param listOfStart []string: list of start operations
@param path map[string]string: currently viewed path
@param send bool: true for search of send without partner, false for receive
@param vcTrace []vct: VCT
@returns string: string with found problems
*/
func findImpossibleCommunication(rs map[infoTime][]infoTime, listOfStart []infoTime, path map[infoTime]infoTime, send bool, vcTrace []vcn) string {
	// fmt.Println(rs)
	if len(listOfStart) == 0 {
		return ""
	}
	start, listOfStart := listOfStart[0], listOfStart[1:]
	found := false
	resString := ""
	for _, end := range rs[start] {
		if !partnerTaken(end, path) {
			path[start] = end
			found = true
			resString += findImpossibleCommunication(rs, listOfStart, path, send, vcTrace)
		}
	}
	// fmt.Println("Path: ", path, found, "\n")
	if !found && isPathPossible(path, vcTrace, send, start) {
		if send {
			resString += "No communication partner for send at "
		} else {
			resString += "No communication partner for receive at "
		}
		resString += start.pos
		if len(path) > 0 {
			resString += " when running the following communication:\n"
			for s, r := range path {
				if send {
					resString += "    " + s.pos + " -> " + r.pos + "\n"
				} else {
					resString += "    " + r.pos + " -> " + s.pos + "\n"
				}
			}
		} else {
			resString += "\n"
		}
		resString += "\n"
	}
	return resString
}

/*
Check if an operation is already in the path
@param receive string: operation
@param map[string]string: path
@return bool, true if operation is already in path, false otherwise
*/
func partnerTaken(receive infoTime, path map[infoTime]infoTime) bool {
	for _, rec := range path {
		if rec == receive {
			return true
		}
	}
	return false
}

/*
Test if a path is valid. To be valid it must be for all Operations in the
path, that all Operations in the same routine before the path mut be
in the path as well
@param path map[string]string: path to test
@param vcTrace []vct: VCT
@param send bool: true, if the operation is send, false if receive
@param s string: string of operation without communication
*/
func isPathPossible(path map[infoTime]infoTime, vcTrace []vcn, send bool, s infoTime) bool {
	for start, end := range path {
		if start == s || end == s {
			return false
		}
		i_start := -1
		i_end := -1
		for j, v := range vcTrace {
			vIt := infoTime{preTime: v.preTime, pos: v.position}
			if start == vIt {
				i_start = j
			}
			if end == vIt {
				i_end = j
			}
		}
		if i_start == -1 || i_end == -1 {
			return false
		}

		for i, v := range vcTrace {
			if i < i_start &&
				v.send == send &&
				v.routine == vcTrace[i_start].routine {
				found := false
				for s, _ := range path {
					vIt := infoTime{preTime: v.preTime, pos: v.position}
					if s == vIt {
						found = true
						continue
					}
				}
				if !found {
					return false
				}
			}
			if i < i_end &&
				v.send == send &&
				v.routine == vcTrace[i_end].routine {
				found := false
				for _, e := range path {
					vIt := infoTime{preTime: v.preTime, pos: v.position}
					if vIt == e {
						found = true
						continue
					}
				}
				if !found {
					return false
				}
			}
		}
	}
	return true
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
					uncompPre := vcUnComparable(preVc, vc.pre)
					uncompPost := vcUnComparable(postVc, vc.post)
					if vc.id == sel.chanId && vc.send && (uncompPre || uncompPost) {
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
Test wether operation with vectorclock vc1 is before vc2
*/
func vcIsBeforeOrConcurrent(vc1, vc2 []int) bool {
	smaller := false
	for i := 0; i < len(vc1); i++ {
		if vc2[i] > vc1[i] {
			return true
		}
		if vc2[i] < vc1[i] {
			smaller = true
		}
	}
	return !smaller
}

/*
Test wether operation with vectorclock vc1 is before vc2
*/
func vcIsBefore(vc1, vc2 []int) bool {
	smaller := false
	for i := 0; i < len(vc1); i++ {
		if vc1[i] > vc2[i] {
			return false
		}
		if vc1[i] < vc2[i] {
			smaller = true
		}
	}
	return smaller
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
