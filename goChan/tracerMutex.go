package goChan

import (
	"sync"
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
Package: goChan
Project: Bachelor Thesis at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Analysis of message passing go programs
*/

/*
tracerMutex.go
Drop in replacements for (rw)mutex and (Try)(R)lock and (Try)(R-)Unlock
*/

/*
Struct to implement a mutex. Is used as drop-in replacement for sync.Mutex
@field mu *sync.Mutex: actual mutex to perform lock and unlock operations
@field id uint32: id of the mutex
*/
type Mutex struct {
	mu       *sync.Mutex
	creation string
	id       uint32
}

/*
Function to create and initialize a new Mutex
*/
func NewMutex() Mutex {
	m := Mutex{mu: &sync.Mutex{}, creation: getPosition(1), id: atomic.AddUint32(&numberOfMutex, 1)}
	return m
}

/*
Function to lock a Mutex.
@receiver *Mutex
*/
func (m *Mutex) Lock() {
	m.t_Lock(false)
}

/*
Function to try-lock a Mutex.
@receiver *Mutex
@return bool: true if lock was successful, false otherwise
*/
func (m *Mutex) TryLock() bool {
	return m.t_Lock(true)
}

/*
Function as helper to perform actual locking of Mutex.
@receiver *Mutex
@param try bool: true if lock is try-lock, false otherwise
@return bool: true if lock was successful, false otherwise
*/
func (m *Mutex) t_Lock(try bool) bool {
	index := getIndex()

	position := getPosition(2)

	if m.mu == nil {
		*m = NewMutex()
	}

	elemIndex := len(traces[index])

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceLock{position: position, timestamp: atomic.AddUint32(&counter, 1),
		lockId: m.id, mutexCreation: m.creation, try: try, read: false, suc: false})
	tracesLock.Unlock()

	res := true
	if try {
		res = m.mu.TryLock()

	} else {
		m.mu.Lock()
	}

	tracesLock.Lock()
	traces[index][elemIndex].(*TraceLock).suc = true
	tracesLock.Unlock()

	return res
}

/*
Function to unlock a Mutex.
@receiver *Mutex
*/
func (m *Mutex) Unlock() {
	index := getIndex()

	position := getPosition(2)

	m.mu.Unlock()

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceUnlock{position: position, timestamp: atomic.AddUint32(&counter, 1),
		lockId: m.id, mutexCreation: m.creation})
	tracesLock.Unlock()
}

/*
Struct to implement a rw-mutex. Is used as drop-in replacement for sync.RWMutex
@field mu *sync.RWMutex: actual rw-mutex to perform lock and unlock operations
@field id uint32: id of the mutex
*/
type RWMutex struct {
	mu       *sync.RWMutex
	creation string
	id       uint32
}

/*
Function to create and initialize a new RWMutex
@return RWMutex: new RWMutex object
*/
func NewRWMutex() RWMutex {
	m := RWMutex{mu: &sync.RWMutex{}, creation: getPosition(1), id: atomic.AddUint32(&numberOfMutex, 1)}
	return m
}

/*
Function to lock a RWMutex.
@receiver *RWMutex
*/
func (m *RWMutex) Lock() {
	m.t_RwLock(false, false)
}

/*
Function to r-lock a RWMutex.
@receiver *RWMutex
*/
func (m *RWMutex) RLock() {
	m.t_RwLock(false, true)
}

/*
Function to try-lock a Mutex.
@receiver *RWMutex
@return bool: true if lock was successful, false otherwise
*/
func (m *RWMutex) TryLock() bool {
	return m.t_RwLock(true, false)
}

/*
Function to try-r-lock a Mutex.
@receiver *RWMutex
@return bool: true if lock was successful, false otherwise
*/
func (m *RWMutex) TryRLock() bool {
	return m.t_RwLock(true, true)
}

/*
Function as helper to perform actual locking of RWMutex.
@receiver *RWMutex
@param try bool: true if lock is try-lock, false otherwise
@param read bool: true if lock is r-lock, false otherwise
@return bool: true if lock was successful, false otherwise
*/
func (m *RWMutex) t_RwLock(try bool, read bool) bool {
	index := getIndex()

	position := getPosition(2)

	if m.mu == nil {
		*m = NewRWMutex()
	}

	elemIndex := len(traces[index])

	tracesLock.Lock()
	traces[index] = append(traces[index],
		&TraceLock{position: position, timestamp: atomic.AddUint32(&counter, 1),
			lockId: m.id, mutexCreation: m.creation, try: try, read: read,
			suc: false})
	tracesLock.Unlock()

	res := true
	if try {
		if read {
			res = m.mu.TryRLock()
		} else {
			res = m.mu.TryLock()
		}
	} else {
		if read {
			m.mu.RLock()
		} else {
			m.mu.Lock()
		}
	}

	tracesLock.Lock()
	traces[index][elemIndex].(*TraceLock).suc = true
	tracesLock.Unlock()

	return res
}

/*
Function to unlock a RWMutex.
@receiver *RWMutex
*/
func (m *RWMutex) Unlock() {
	m.t_Unlock(false)
}

/*
Function to r-unlock a RWMutex.
@receiver *RWMutex
*/
func (m *RWMutex) RUnlock() {
	m.t_Unlock(true)
}

/*
Function as helper to perform actual unlock on RWMutex
@receiver *RWMutex
@param read bool: true if it is a r-unlock, false otherwise
*/
func (m *RWMutex) t_Unlock(read bool) {
	index := getIndex()

	position := getPosition(2)

	if read {
		m.mu.RUnlock()
	} else {
		m.mu.Unlock()
	}

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceUnlock{position: position, timestamp: atomic.AddUint32(&counter, 1),
		lockId: m.id, mutexCreation: m.creation})
	tracesLock.Unlock()
}
