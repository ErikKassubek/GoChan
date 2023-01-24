package goChan

import "fmt"

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
analyzer.go
Main functions to start the analyzer
*/

var running bool = false

/*
Main function to run the analyzer. The running of the analyzer locks
tracesLock for the total duration of its runtime, to prevent go-routines,
that are still running when the main function terminated (and therefore would
normally also be terminated) to alter the trace.
*/
func RunAnalyzer() {
	if running {
		return
	}
	running = true
	tracesLock.Lock()

	// analyze the trace for potential deadlocks including only mutexe based
	// on
	_, resString := analyzeMutexDeadlock()
	// res = res || r

	vcTrace, ok, creations := buildVectorClockChan()

	// res = res || ok
	danglingEventChans := ""
	for _, id := range creations {
		danglingEventChans += "\n    " + fmt.Sprint(id)
	}
	if ok {
		resString = append(resString, fmt.Sprintf("Found dangling Events for Channels created at:  %s", danglingEventChans))
	}

	r, rs := checkForNonEmptyChan(vcTrace)
	resString = append(resString, rs...)

	// fmt.Println(vcTrace)
	if ok || r {
		rs = findAlternativeCommunication(vcTrace)
		if len(rs) == 0 {
			resString = append(resString, "\nNo possible communication found!")
		}
		resString = append(resString, rs...)
	}

	_, rs = checkForPossibleSendToClosed(vcTrace)
	// res = res || r
	resString = append(resString, rs...)

	tracesLock.Unlock()

	// print res Strings
	for _, prob := range resString {
		fmt.Println("##@@##" + prob + "##@@##")
	}
}
