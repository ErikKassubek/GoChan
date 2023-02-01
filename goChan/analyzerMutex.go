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
analyzerMutex.go
Analyze the trace to check for deadlocks containing only (rw-)mutexe based on
the UNDEAD algorithm.
*/

var lockGraph [][]dependency
var noDep int = 0

/*
Main function to analyze the trace to find potential deadlocks containing only
(rw-)mutexe.
@return bool: true, if a potential deadlock was found, false otherwise
@return string: list of messages for the found potential deadlocks
*/
func analyzeMutexDeadlock() (bool, []string) {
	res := false
	resString := make([]string, 0)

	//check for double locking
	rd, sd := checkForDoubleLocking()
	res = res || rd
	resString = append(resString, sd...)

	// build the graph
	r, s := buildGraph()
	res = res || r
	resString = append(resString, s...)

	// check for circular deadlocks if at least two routines and two
	// Dependencies exist
	if len(traces) > 1 && noDep > 1 {
		r, s := findPotentialMutexDeadlocksCirc()
		res = res || r
		resString = append(resString, s...)
	}

	return res, resString
}

/*
Check for double locking
*/
func checkForDoubleLocking() (bool, []string) {
	r := false
	res := make([]string, 0)
	for _, trace := range traces {
		switch a := trace[len(trace)-1].(type) {
		case *TraceLock:
			found := false
			for i := len(trace) - 2; i >= 0; i-- {
				switch b := trace[i].(type) {
				case *TraceLock:
					if a.lockId == b.lockId && (!a.read || !b.read) {
						r = true
						res = append(res, fmt.Sprintf("Found double locking:\n    %s -> %s", b.position, a.position))
						found = true
					}
				case *TraceUnlock:
					if a.lockId == b.lockId {
						found = true
					}
				}
				if found {
					break
				}
			}
		}
	}
	return r, res
}

/*
Build a lock-graph from the trace.
@return bool: true, if non released lock was found
@return []string]: info about non released lock
*/
func buildGraph() (bool, []string) {
	lockGraph = make([][]dependency, len(traces))
	res := false
	resString := make([]string, 0)

	for index, trace := range traces {
		lockGraph[index] = make([]dependency, 0)
		currentHoldLocks := make([]TraceElement, 0)
		for _, elem := range trace {
			switch e := elem.(type) {
			case *TraceLock:
				// add dependency to graph and lock to currentHoldLocks
				lockGraph[index] = append(lockGraph[index], newDependency(e, currentHoldLocks))
				currentHoldLocks = append(currentHoldLocks, e)
				noDep++
			case *TraceUnlock: // remove lock from currently hold locks
				for i := len(currentHoldLocks) - 1; i >= 0; i-- {
					if currentHoldLocks[i].(*TraceLock).lockId == e.lockId {
						currentHoldLocks = append(currentHoldLocks[:i], currentHoldLocks[i+1:]...)
					}
				}
			}
		}
		// check if a lock was still locked at the end
		for _, l := range currentHoldLocks {
			res = false
			resString = append(resString, fmt.Sprintf("Locked Mutex Not Freed (May be caused by occurring deadlock):\n  %s", l.(*TraceLock).position))
		}
	}

	return res, resString
}

// detect runs the detection for loops in the lock trees
//
//	Returns:
//	 nil
func findPotentialMutexDeadlocksCirc() (bool, []string) {
	res := false
	resString := make([]string, 0)

	// visiting gets set to index of the routine on which the search for circles is started
	var visiting int

	// A stack is used to represent the currently explored path in the lock trees.
	// A dependency is added to the path by pushing it on top of the stack.
	stack := newDepStack()

	// If a routine has been used as starting routine of a cycle search, all
	// possible paths have already been explored and therefore have no circle.
	// The dependencies in this routine can therefor be ignored for the rest
	// of the search.
	// They can also be temporarily ignored, if a dependency of this routine
	// is already in the path which is currently explored
	isTraversed := make([]bool, len(lockGraph))

	// traverse all routines as starting routine for the loop search
	for i, routine := range lockGraph {
		visiting = i

		// traverse all dependencies of the given routine as starting routine
		// for potential paths
		for _, dep := range routine {

			// push the dependency on the stack as first element of the currently
			// explored path
			stack.push(&dep, i)

			// start the depth-first search to find potential circular paths
			r, rs := dfs(&stack, visiting, &isTraversed)
			res = res || r
			resString = append(resString, rs...)

			// remove dep from the stack
			stack.pop()
		}
		isTraversed[i] = true
	}

	return res, resString
}

/*
dfs runs the recursive depth-first search.
Only paths which build a valid chain are explored.
After a new dependency is added to the currently explored path, it is checked,
if the path forms a circle.
@param stack (*depStack): stack witch represent the currently explored path
@param visiting int: index of the routine of the first element in the currently explored path
@param isTraversed (*([]bool)): list which stores which routines have already been traversed (either as starting routine or as a routine which already has a dep in the current path)
@return bool: true if a potential deadlock was detected, false otherwise
@return string: description of the potential deadlock
*/
func dfs(stack *depStack, visiting int, isTraversed *([]bool)) (bool, []string) {
	res := false
	resString := make([]string, 0)

	// Traverse through all routines to find the potential next step in the path.
	// Routines with index <= visiting have already been used as starting routine
	// and therefore don't have to been considered again.
	for i := visiting + 1; i < len(lockGraph); i++ {
		routine := lockGraph[i]

		// continue if the routine has already been traversed
		if (*isTraversed)[i] {
			continue
		}

		// go through all dependencies of the current routine
		for j := 0; j < len(routine); j++ {
			dep := routine[j]
			// check if adding dep to the stack would still be a valid path
			if isChain(stack, &dep, i) {
				// check if adding dep to the stack would lead to a cycle
				if isCycleChain(stack, &dep, i) {
					// report the found potential deadlock
					stack.push(&dep, j)
					res = true
					resString = append(resString, getDeadlockMessage(stack))
					stack.pop()
				} else { // the path is not a cycle yet
					// add dep to the current path
					stack.push(&dep, i)
					(*isTraversed)[i] = true

					// call dfs recursively to traverse the path further
					r, rs := dfs(stack, visiting, isTraversed)
					res = res || r
					resString = append(resString, rs...)

					// dep did not lead to a cycle in the lock trees.
					// It is removed to explore different paths
					stack.pop()
					(*isTraversed)[i] = false
				}
			}
		}
	}
	return res, resString
}

// isCain checks if adding dep to the current path represented by stack is
// still a valid path.
//
//	A valid path contains the same dependency only once and contains the same
//	lock only once. A path is also not valid if there exist two locks in the
//
// holdings sets of two different dependencies in the path, such that the locks
// are equal. This would be a gate lock situation. For RW-Locks this is not
// true if both of the locks were acquired with RLock, because RLocks don't
// have to work as gate locks
//
//	Args:
//	 stack (*depStack): stack representing the current path
//	 dep (*dependency): dependency for which it should be checked if it can be
//	  added to the path
//	 routineIndex (int): index of the routine the dependency is from
//	Returns:
//	 (bool): true if dep can be added to the current path, false otherwise
func isChain(stack *depStack, dep *dependency, routineIndex int) bool {
	// the mutex of the depEntry at the top of the stack mut be in the
	// holding set of dep
	found := false
	for _, mutexInHs := range dep.holdingSet {
		if mutexInHs.(*TraceLock).lockId == stack.top.depEntry.mu.(*TraceLock).lockId {
			// if mutexInHs is read, the mutex at the top of the stack can not also be read
			if !(mutexInHs.(*TraceLock).read && stack.top.depEntry.mu.(*TraceLock).read) {
				found = true
				break
			}
		}
	}
	if !found {
		return false
	}

	for c := stack.stack.next; c != nil; c = c.next {
		// no two dependencies in the stack can be equal
		if c.depEntry == dep {
			return false
		}

		// If two holding sets contain the same mutex they both have to be rLock
		// (gate lock)
		for i := 0; i < len(dep.holdingSet); i++ {
			for j := 0; j < len(c.depEntry.holdingSet); j++ {
				lockInDepHs := dep.holdingSet[i]
				lockInCHoldingSet := c.depEntry.holdingSet[j]
				if lockInDepHs.(*TraceLock).lockId == lockInCHoldingSet.(*TraceLock).lockId {
					if !(lockInCHoldingSet.(*TraceLock).read && lockInDepHs.(*TraceLock).read) {
						return false
					}
				}
			}
		}
	}

	return true
}

// isCycleCain checks if adding a dependency dep to the current path represented
// by stack would lead to a cyclic chain, meaning the lock mu of dep is in the
// holding set of the first dependency in the path. This would indicate a possible
// deadlock situation. With RW-locks it is possible, that a cyclic path
// does not indicate a potential deadlock. In this case, the function assumes,
// that the path does not create a valid cyclic chain.
//
//	isCycleChain assumes, that adding dep to the path results to a valid path
//	(see isChain)
//
// Args:
//
//	stack (*depStack): stack representing the current path
//	dep (*dependency): dependency for which it should be checked if adding dep
//	 to the path would lead to a cyclic path
//	routineIndex (int): index of the routine from which dep originated
//
// Returns:
//
//	(bool): true if dep can be added to the current path to create a valid cyclic
//	 chain, false if the path is no cycle, or it contains RW-lock with which
//	 the cycle does not indicate a deadlock
func isCycleChain(dStack *depStack, dep *dependency, routineIndex int) bool {
	// the mutex dep must be in the holding set of the depEntry at the bottom of
	// the stack
	found := false
	for _, mutexInHs := range dStack.stack.next.depEntry.holdingSet {
		if mutexInHs.(*TraceLock).lockId == dep.mu.(*TraceLock).lockId {
			// if mutexInHs is read, the mutex at the top of the stack can not also be read
			if !(mutexInHs.(*TraceLock).read && dep.mu.(*TraceLock).read) {
				found = true
				break
			}
		}
	}
	return found
}

/*
Function to get the string for a potential cyclic deadlock
@param stack *depStack: stack
@return string: message
*/
func getDeadlockMessage(stack *depStack) string {
	message := "Potential Cyclic Mutex Locking:\n"
	for cl := stack.stack.next; cl != nil; cl = cl.next {
		lock := cl.depEntry.mu
		switch l := lock.(type) {
		case *TraceLock:
			hs := ""
			for _, h := range cl.depEntry.holdingSet {
				switch h_t := h.(type) {
				case *TraceLock:
					hs += "    " + h_t.position + "\n"
				}
			}
			message += fmt.Sprintf("Lock: %s\n  Hs:\n%s", l.position, hs)
		}
	}
	return message
}
