// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"sync"
)

var ErrNoCopyFunc = errors.New("store: copyFunc must not be nil")

// Reader lets the consumer read entries from [Store].
type Reader[T any] interface {
	Read(key string) (T, bool)
}

// Writer lets the consumer write entries to [Store].
type Writer[T any] interface {
	Write(key string, data T)
	Delete(key string)
}

// Store is a thread safe in-memory store that can be used to
// read and write data. Mind that the store
// does not perform any validation on the inputs.
type Store[T any] struct {
	mutex    sync.RWMutex
	store    map[string]T
	copyFunc func(T) T
}

// NewStore returns a ready for use [Store].
func NewStore[T any](copyFunc func(T) T) (*Store[T], error) {
	if copyFunc == nil {
		return nil, ErrNoCopyFunc
	}
	return &Store[T]{
		store:    make(map[string]T),
		copyFunc: copyFunc,
	}, nil
}

// MustNewStore returns a ready for use [Store].
// It panics if copyFunc is nil.
func MustNewStore[T any](copyFunc func(T) T) *Store[T] {
	store, err := NewStore(copyFunc)
	if err != nil {
		panic(err)
	}
	return store
}

// Read retrieves an entry from the [Store].
func (s *Store[T]) Read(key string) (T, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.store[key]
	if ok {
		return s.copyFunc(data), ok
	}
	var t T
	return t, false
}

// Write sets and entry to the [Store].
// If the entry exists it is overwritten.
func (s *Store[T]) Write(key string, data T) {
	d := s.copyFunc(data)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.store[key] = d
}

// Delete removes an entry from the [Store].
func (s *Store[T]) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.store, key)
}

// Len returns the number of entries in the [Store].
func (s *Store[T]) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.store)
}
