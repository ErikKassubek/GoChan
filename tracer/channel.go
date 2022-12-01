package tracer

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
channel.go
Drop in replacements for channels and send and receive functions
*/

type Message[T any] struct {
	info            T
	sender          uint32
	senderTimestamp uint32
}

// create a message object
func BuildMessage[T any](info T) Message[T] {
	index := getIndex()
	return Message[T]{info: info, sender: index, senderTimestamp: atomic.LoadUint32(&counter)}
}

// get message info
func (m *Message[T]) GetInfo() T {
	return m.info
}

// struct to implement a drop in replacement for a channel
type Chan[T any] struct {
	c  chan Message[T]
	id int
}

// create a new channel with type T and size size, drop in replacement for
// make(chan T, size), size = 0 for unbuffered channel
func NewChan[T any](size int) Chan[T] {
	ch := Chan[T]{c: make(chan Message[T], size), id: int(atomic.AddUint32(&numberOfChan, 1))}

	return ch
}

// get the channel, manly for switch
func (ch *Chan[T]) GetChan() chan Message[T] {
	return ch.c
}

// get the id of the channel
func (ch *Chan[T]) GetId() int {
	return ch.id
}

type PreObj struct {
	id      int
	receive bool
}

// get an object containing the id and send/receive
func (ch *Chan[T]) GetIdPre(receive bool) PreObj {
	return PreObj{id: ch.id, receive: receive}
}

// drop in replacement for sending val on channel c
func (ch *Chan[T]) Send(val T) {
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)

	// add pre event to tracer
	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePre{timestamp: timestamp,
		chanId: ch.id, send: true})
	tracesLock.Unlock()

	ch.c <- Message[T]{
		info:            val,
		sender:          index,
		senderTimestamp: timestamp,
	}

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePost{chanId: ch.id, send: true,
		SenderId: index, timestamp: atomic.AddUint32(&counter, 1)})
	tracesLock.Unlock()
}

// drop in replacement for receiving value on channel chan and returning value
func (ch *Chan[T]) Receive() T {
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePre{timestamp: timestamp,
		chanId: ch.id, send: false})
	tracesLock.Unlock()

	res := <-ch.c

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePost{
		timestamp: atomic.AddUint32(&counter, 1), chanId: ch.id, send: false,
		SenderId: res.sender, senderTimestamp: res.senderTimestamp})
	tracesLock.Unlock()

	return res.info
}

// drop in replacement for closing a channel
func (ch *Chan[T]) Close() {
	index := getIndex()
	timestamp := atomic.AddUint32(&counter, 1)

	close(ch.c)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceClose{timestamp: timestamp,
		chanId: ch.id})
	tracesLock.Unlock()
}
