package main

import "fmt"

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
analyzer.go
Main functions to start the analyzer
*/

/*
Main function to run the analyzer. The running of the analyzer locks
tracesLock for the total duration of its runtime, to prevent go-routines,
that are still running when the main function terminated (and therefore would
normally also be terminated) to alter the trace.
*/
func RunAnalyzer() {
	fmt.Print("Start Programm analysis...\n\n")
	tracesLock.Lock()

	res := false
	resString := make([]string, 0)

	// analyze the trace for potential deadlocks including only mutexe based
	// on
	r, rs := analyzeMutexDeadlock()
	res = res || r
	resString = append(resString, rs...)

	ok, c := checkForDanglingEvents()
	res = res || ok
	vcTrace := buildVectorClockChan(c)

	r, rs = checkForNonEmptyChan(vcTrace)
	res = res || r
	resString = append(resString, rs...)

	if ok && !r {
		resString = append(resString, "Found dangling Events")
	} else if !ok && r {
		resString = append(resString, "Found non-empty Channel")
	} else if ok && r {
		resString = append(resString, "Found dangling Events and non-empty Channel")
	}
	// fmt.Println(vcTrace)
	if ok || r {
		rs = findAlternativeCommunication(vcTrace)
		resString = append(resString, rs...)
	}

	r, rs = checkForImpossibleSelectStatements(vcTrace)
	res = res || r
	resString = append(resString, rs...)

	tracesLock.Unlock()

	// print res Strings
	if !res {
		fmt.Println("No Problems Detected")
	} else {
		fmt.Print("Problems Detected\n\n")
		for _, prob := range resString {
			fmt.Println(prob)
			fmt.Print("\n")
		}
	}
}
