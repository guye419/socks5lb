/**
 * File: pool.go
 * Author: Ming Cheng<mingcheng@outlook.com>
 *
 * Created Date: Tuesday, June 21st 2022, 6:03:26 pm
 * Last Modified: Thursday, July 7th 2022, 6:47:39 pm
 *
 * http://www.opensource.org/licenses/MIT
 */

package socks5lb

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type Pool struct {
	backends map[string]*Backend
	current  uint64
	lock     sync.Mutex
}

func (b *Pool) Add(backend *Backend) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.backends[backend.Addr] != nil {
		return fmt.Errorf("%v is already exists, remove it first", backend.Addr)
	}

	b.backends[backend.Addr] = backend
	return
}

func (b *Pool) Remove(addr string) (err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.backends, addr)
	return
}

// AllHealthy returns all healthy backends
func (b *Pool) AllHealthy() (backends []*Backend) {
	for _, v := range b.backends {
		if v.Alive() {
			backends = append(backends, v)
		}
	}

	return
}

func (b *Pool) NextIndex() int {
	return int(atomic.AddUint64(&b.current, uint64(1)) % uint64(len(b.backends)))
}

// Next returns the next index in the pool if there is one available
// Only supports round-robin operations by default
func (b *Pool) Next() *Backend {

	// return healthy backends first
	backends := b.AllHealthy()
	log.Tracef("found all %d available backends", len(backends))

	// can not found any backends available
	if len(backends) <= 0 {
		return nil
	}

	// loop entire backends to find out an Alive backend
	next := b.NextIndex()
	// start from next and move a full cycle
	l := len(backends) + next

	for i := next; i < l; i++ {
		// take an index by modding
		idx := i % len(backends)

		// if we have an alive backend, use it and store if its not the original one
		if backends[idx].Alive() {
			if i != next {
				atomic.StoreUint64(&b.current, uint64(idx))
			}

			return backends[idx]
		}
	}

	return nil
}

// Check if we have an alive backend
func (b *Pool) Check() {
	for _, b := range b.backends {
		err := b.Check()
		if err != nil {
			log.Errorf("check backend %s is failed, error %v", b.Addr, err)
		} else {
			log.Debugf("check backend %s is successful", b.Addr)
		}
	}
}

var (
	instance *Pool
	once     sync.Once
)

// NewPool instance for a new Pools instance
func NewPool() *Pool {
	once.Do(func() {
		instance = &Pool{
			backends: make(map[string]*Backend),
		}
	})

	return instance
}
