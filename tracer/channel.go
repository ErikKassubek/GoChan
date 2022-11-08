package tracer

import "sync"

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
channel.go
Drop in replacements for channels and send and receive functions
*/

var numberOfChan int = 0

// struct to implement a drop in replacement for a channel
type Chan[T any] struct {
	c               chan T
	id              int
	sender          []int
	senderTimestamp []int
	lock            sync.Mutex
}

// create a new channel with type T and size size, drop in replacement for
// make(chan T, size), size = 0 for unbuffered channel
func NewChan[T any](size int) Chan[T] {
	ch := Chan[T]{c: make(chan T, size), id: numberOfChan, sender: make([]int, 0)}
	numberOfChan++
	return ch
}

// get the channel, manly for switch
func (ch *Chan[T]) GetChan() chan T {
	return ch.c
}

// get the id of the channel
func (ch *Chan[T]) GetId() int {
	return ch.id
}

// drop in replacement for sending val on channel c
func (ch *Chan[T]) Send(val T) {
	ch.lock.Lock()
	index := getIndex()
	counter[index]++

	// add pre event to tracer
	traces[index] = append(traces[index], &TracePre{chanId: ch.id, send: true})

	ch.sender = append(ch.sender, index)
	ch.senderTimestamp = append(ch.senderTimestamp, counter[index])
	ch.c <- val

	traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: true,
		SenderId: index, timestamp: counter[index]})
	ch.lock.Unlock()
}

// drop in replacement for receiving value on channel chan and returning value
func (ch *Chan[T]) Receive() T {
	ch.lock.Lock()
	index := getIndex()

	counter[index]++
	traces[index] = append(traces[index], &TracePre{chanId: ch.id, send: false})

	res := <-ch.c
	senderId := ch.sender[0]
	ch.sender = ch.sender[1:]
	senderTimestamp := ch.senderTimestamp[0]
	ch.senderTimestamp = ch.senderTimestamp[1:]

	traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: false,
		SenderId: senderId, timestamp: senderTimestamp})
	ch.lock.Unlock()
	return res
}

// drop in replacement for closing a channel
func (ch *Chan[T]) Close() {
	ch.lock.Lock()
	index := getIndex()
	close(ch.c)
	traces[index] = append(traces[index], &TraceClose{ch.id})
	ch.lock.Unlock()
}
