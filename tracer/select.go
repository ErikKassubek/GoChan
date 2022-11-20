package tracer

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

// add before select,
// def is true if the select has a default path
func PreSelect(def bool, channels ...int) {
	index := getIndex()
	counter[index]++

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePreSelect{channels, def})
	tracesLock.Unlock()
}

// add at begging of select block
func (ch *Chan[T]) PostSelect(receive bool, message Message[T]) {
	index := getIndex()
	counter[index]++

	if receive {
		tracesLock.Lock()
		traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: false,
			SenderId: message.sender, timestamp: message.senderTimestamp})
		tracesLock.Unlock()
	} else {
		tracesLock.Lock()
		counterLock.RLock()
		traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: true,
			SenderId: index, timestamp: counter[index]})
		counterLock.RUnlock()
		tracesLock.Unlock()
	}
}

// add to default statement of select
func PostDefault() {
	index := getIndex()
	counter[index]++

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceDefault{})
	tracesLock.Unlock()
}
