package goChan

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
analyzerMutexStack.go
Implementation of a helper type to realize a chain stack.
It is implemented as a double linked list, but it is only possible to remove the
last element (top element on the stack) and add an element at the end of the list
(push on top of the stack)
*/

// ============ stack element ============

// struct to implement a stack element
type stackElement struct {
	// dependency represented by the stack element
	depEntry *dependency
	// index value of the linkedIndex, is set to the index of the routine
	index int
	// pointer to the previous stack element
	prev *stackElement
	// pointer to the next stack element
	next *stackElement
}

/*
Function to create a new chainList
@param dep (*dependency): dependency which is represented by the stack element
@param i (int): index of the routine which created dep
@return stackElement: element for the stack
*/
func newStackElement(dep *dependency, i int) stackElement {
	return stackElement{
		depEntry: dep,
		index:    i,
		prev:     nil,
		next:     nil,
	}
}

// ============ stack ============

// stack for the dependencies
type depStack struct {
	// pointer to the bottom element of the stack
	stack *stackElement
	// pointer to the top element of the stack
	top *stackElement
}

/*
Function to create a new stack
@return depStack: the dependency stack
*/
func newDepStack() depStack {
	cl := newStackElement(nil, -1)

	// set the first element of the stack to an empty stack element
	c := depStack{
		stack: &cl,
	}

	// the first element in the stack is the only and therefore also last element
	c.top = c.stack

	return c
}

/*
Function to push a new dependency to the stack
@param dep (*dependency): dependency to put on the stack
@param index (int): index of the routine which created the dependency
*/
func (s *depStack) push(dep *dependency, index int) {
	// create the new element
	cl := newStackElement(dep, index)
	// add it to the stack
	s.top.next = &cl
	// reset the pointers of the previous element and the pointer to the top element
	cl.prev = s.top
	s.top = &cl
}

/*
Function to remove the top element from stack
*/
func (s *depStack) pop() {
	// do nothing if the stack is empty (has only on empty default element)
	if s.top == s.stack {
		return
	}

	// reroute the pointer to remove the top stack element
	s.top.prev.next = s.top.next
	s.top = s.top.prev
}
