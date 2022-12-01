package tracer

import (
	"sync/atomic"

	"github.com/petermattis/goid"
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
spawn.go
Drop in replacements to create and start a new go routine
*/

/*
Function to call before the creation of a new routine in the old routine
@return id of the new routine
*/
func SpawnPre() uint32 {
	nr := atomic.AddUint32(&numberRoutines, 1)
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceSignal{timestamp: timestamp, routine: nr})
	traces = append(traces, make([]TraceElement, 0))
	tracesLock.Unlock()

	return nr
}

/*
Function to call after the creation of a new routine in the new routine
@param numRut uint32: id of the new routine
*/
func SpawnPost(numRut uint32) {
	id := goid.Get()

	timestamp := atomic.AddUint32(&counter, 1)

	routineIndexLock.Lock()
	routineIndex[id] = numRut
	routineIndexLock.Unlock()

	tracesLock.Lock()
	traces[numRut] = append(traces[numRut], &TraceWait{timestamp: timestamp, routine: numRut})
	tracesLock.Unlock()
}
