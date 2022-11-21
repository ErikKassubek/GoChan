package tracer

import (
	"sync"
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
mutex.go
Drop in replacements for (rw)mutex and (Try)(R)lock and (Try)(R-)Unlock
*/

var numberOfMutex int = 0
var numberOfMutexLock sync.Mutex

// Mutex
type Mutex struct {
	mu *sync.Mutex
	id int
}

// create mutex
func NewLock() *Mutex {
	numberOfMutexLock.Lock()
	m := Mutex{mu: &sync.Mutex{}, id: numberOfMutex}
	numberOfMutex++
	numberOfMutexLock.Unlock()
	return &m
}

// lock a mutex
func (m *Mutex) Lock() {
	m.t_Lock(false)
}

// try to lock a mutex
func (m *Mutex) TryLock() bool {
	return m.t_Lock(true)
}

// helper for lock on mutex
func (m *Mutex) t_Lock(try bool) bool {
	index := getIndex()

	counterLock.RLock()
	counter[index]++
	counterLock.RUnlock()

	res := false
	if try {
		res = m.mu.TryLock()
	}

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceLock{lockId: m.id, try: try, read: false, suc: res})
	tracesLock.Unlock()

	return res
}

// unlock mutex
func (m *Mutex) Unlock() {
	index := getIndex()

	counterLock.RLock()
	counter[index]++
	counterLock.RUnlock()

	m.mu.Unlock()

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceUnlock{lockId: m.id})
	tracesLock.Unlock()
}

// create RWutex
func NewRWLock() *RWMutex {
	numberOfMutexLock.Lock()
	m := RWMutex{mu: &sync.RWMutex{}, id: numberOfMutex}
	numberOfMutex++
	numberOfMutexLock.Unlock()
	return &m
}

// RW-Mutex
type RWMutex struct {
	mu *sync.RWMutex
	id int
}

// lock rwMutex
func (m *RWMutex) Lock() {
	m.t_RwLock(false, false)
}

// tryLock rwMutex
func (m *RWMutex) RLock() {
	m.t_RwLock(false, true)
}

// trylock rwMutex
func (m *RWMutex) TryLock() bool {
	return m.t_RwLock(true, false)
}

// tryRlock rwMutex
func (m *RWMutex) TryRLock() bool {
	return m.t_RwLock(true, true)
}

// helper function to lock rwMutex
func (m *RWMutex) t_RwLock(try bool, read bool) bool {
	index := getIndex()

	counterLock.RLock()
	counter[index]++
	counterLock.RUnlock()

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
	traces[index] = append(traces[index], &TraceLock{lockId: m.id, try: try, read: read, suc: res})
	tracesLock.Unlock()

	return res
}

// unlock rwMutex
func (m *RWMutex) Unlock() {
	m.t_Unlock(false)
}

// rUnlock rwMutex
func (m *RWMutex) RUnlock() {
	m.t_Unlock(true)
}

// helper function to unlock rwMutex
func (m *RWMutex) t_Unlock(read bool) {
	index := getIndex()

	counterLock.RLock()
	counter[index]++
	counterLock.RUnlock()

	if read {
		m.mu.RUnlock()
	} else {
		m.mu.Unlock()
	}

	tracesLock.Lock()
	traces[index] = append(traces[index], &TraceUnlock{lockId: m.id})
	tracesLock.Unlock()
}
