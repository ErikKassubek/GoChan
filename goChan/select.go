package goChan

import (
	"sync/atomic"
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
select.go
Drop in replacements for select
*/

/*
Function to add before a select statement.
@param def bool: true if the select has a default statement, false otherwise
@param channels ...PreObj: list of PreObj to store the cases
@return nil
*/
func PreSelect(def bool, channels ...PreObj) {
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePreSelect{timestamp: timestamp,
		chanIds: channels, def: def})
	tracesLock.Unlock()
}

/*
Function to add at the beginning of a select case body.
&param receive bool: true, if the case was started with a receive false if with a send
@param message Message[T]: message wich was send over the channel
@return nil
*/
func (ch *Chan[T]) Post(receive bool, message Message[T]) {
	index := getIndex()
	timestamp := atomic.AddUint32(&counter, 1)

	if receive {
		tracesLock.Lock()
		traces[index] = append(traces[index], &TracePost{timestamp: timestamp,
			chanId: ch.id, send: false,
			senderId: message.sender, senderTimestamp: message.senderTimestamp})
		tracesLock.Unlock()
	} else {
		tracesLock.Lock()
		traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: true,
			senderId: index, timestamp: timestamp})
		tracesLock.Unlock()
	}
}

/*
Function to add at the beginning of a select default body.
@return nil
*/
func PostDefault() {
	index := getIndex()
	timestamp := atomic.AddUint32(&counter, 1)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceDefault{timestamp: timestamp})
	tracesLock.Unlock()
}
