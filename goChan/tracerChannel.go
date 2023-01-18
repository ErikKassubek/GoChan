package goChan

import (
	"sync/atomic"
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
tracerChannel.go
Drop in replacements for channels and send and receive functions
*/

/*
Struct for message to send by Chan object
@field info T: actual message to send
@field sender uint32: id of the sender
@field senderTimestamp: timestamp when the message was send
*/
type Message[T any] struct {
	info            T
	sender          uint32
	senderTimestamp uint32
	open            bool
}

/*
Funktion to create a message.
This function is mainly used, when sending a message in a select statement
@param info T: info to send
@return Message[T]: message object wich can be send
*/
func BuildMessage[T any](info T) Message[T] {
	index := getIndex()
	return Message[T]{info: info, sender: index, senderTimestamp: atomic.LoadUint32(&counter), open: true}
}

/*
Function to get the info field from a message.
@receiver *Message[T]
@return T: info field of the message
*/
func (m *Message[T]) GetInfo() T {
	return m.info
}

/*
Function to get the info field from a message which was received by a, ok := <- c.
@receiver *Message[T]
@return T: info field of the message
@return bool: true, if channel was open, false otherwise
*/
func (m *Message[T]) GetInfoOk() (T, bool) {
	return m.info, m.open
}

/*
Struct to implement a drop in replacement for a channel
@field c chan Message[T]: channel to send a message
@field id uint32: id for the channel
@field capacity: max size of the channel
@field noSend: number of completed sends on the channel
@field noRec: number of completed receives on the channel
*/
type Chan[T any] struct {
	c        chan Message[T]
	id       uint32
	creation string
	capacity int
	noSend   int
	noRec    int
	open     bool
}

/*
Function to create a new channel object. This object can be used as a drop in
replacement for a chan T.
@param size int: size of the channel, 0 for un-buffered channel
@return Chan[T]: channel object
*/
func NewChan[T any](size int) Chan[T] {
	id := atomic.AddUint32(&numberOfChan, 1)
	ch := Chan[T]{c: make(chan Message[T], size),
		id: id, creation: getPosition(1), capacity: size, noSend: 0, noRec: 0, open: true}

	chanSizeLock.Lock()
	chanSize[id] = size
	chanSizeLock.Unlock()

	return ch
}

/*
Getter for the chan field of a Chan object.
@receiver *Chan[T]
@return chan Message[T]: chan field of channel
*/
func (ch *Chan[T]) GetChan() chan Message[T] {
	return ch.c
}

/*
Getter fir the id field of a Chan object
@receiver *Chan[T]
@return uint32: id of the Chan
*/
func (ch *Chan[T]) GetId() uint32 {
	return ch.id
}

/*
Struct to save a Chan with id. Used to store channel cases in select.
@field id uint32: id of the Chan
@field chanCreation string: pos of the creation of the chan
@field receive bool: true, if the select case is a channel receive, false if it is a send
*/
type PreObj struct {
	id           uint32
	chanCreation string
	receive      bool
}

/*
Function to create a PreObj object from a Chan used with select.
@param receive bool: true, if the select case is a channel receive, false if it is a send
@return PreObj: the created preObj object
*/
func (ch *Chan[T]) GetIdPre(receive bool) PreObj {
	return PreObj{id: ch.id, chanCreation: ch.creation, receive: receive}
}

/*
Function as drop-in replacements for ch.c <- T.
@receiver: *Chan[T]
@param: val T: message to send over the channel
*/
func (ch *Chan[T]) Send(val T) {
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)
	position := getPosition(1)

	// add pre event to tracer
	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePre{position: position, timestamp: timestamp,
		chanId: ch.id, chanCreation: ch.creation, send: true})
	tracesLock.Unlock()

	ch.c <- Message[T]{
		info:            val,
		sender:          index,
		senderTimestamp: timestamp,
		open:            true,
	}

	ch.noSend++

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePost{position: position, chanId: ch.id, chanCreation: ch.creation, send: true,
		senderId: index, timestamp: atomic.AddUint32(&counter, 1), noComs: ch.noSend})
	tracesLock.Unlock()
}

/*
Function as drop-in replacement for <-ch.c.
@receiver: *Chan[T]
@return T: received value
*/
func (ch *Chan[T]) Receive() T {
	index := getIndex()

	timestamp := atomic.AddUint32(&counter, 1)
	position := getPosition(1)

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePre{position: position,
		timestamp: timestamp, chanId: ch.id, chanCreation: ch.creation, send: false})
	tracesLock.Unlock()

	res := <-ch.c

	//do not record post receive on closed channel
	if res.senderTimestamp == 0 {
		return res.info
	}

	ch.noRec++

	tracesLock.Lock()
	traces[index] = append(traces[index], &TracePost{position: position,
		timestamp: atomic.AddUint32(&counter, 1), chanId: ch.id,
		chanCreation: ch.creation, send: false,
		senderId: res.sender, senderTimestamp: res.senderTimestamp, noComs: ch.noRec})
	tracesLock.Unlock()

	return res.info
}

/*
Function as drop-in replacement for a, ok := <-ch.c.
@receiver: *Chan[T]
@return T: received value
*/
func (ch *Chan[T]) ReceiveOk() (T, bool) {
	res := ch.Receive()
	return res, ch.open
}

/*
Function as drop-in replacement for closing a channel.
*/
func (ch *Chan[T]) Close() {
	index := getIndex()
	timestamp := atomic.AddUint32(&counter, 1)
	position := getPosition(1)

	close(ch.c)

	ch.open = false

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceClose{position: position, timestamp: timestamp,
		chanId: ch.id, chanCreation: ch.creation})
	tracesLock.Unlock()
}
