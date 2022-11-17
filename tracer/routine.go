package tracer

import (
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

// call before creating routine
func SpawnPre() int {
	numberRoutines++
	index := getIndex()

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceSignal{numberRoutines})
	traces = append(traces, make([]TraceElement, 0))
	tracesLock.Unlock()

	counterLock.Lock()
	counter[index]++
	counter = append(counter, 0)
	counterLock.Unlock()

	return numberRoutines
}

// call in newly created routine
func SpawnPost(numRut int) {
	id := goid.Get()
	routineIndexLock.Lock()
	routineIndex[id] = numRut
	routineIndexLock.Unlock()

	tracesLock.Lock()
	traces[numRut] = append(traces[numRut], &TraceWait{numRut})
	tracesLock.Unlock()
}
