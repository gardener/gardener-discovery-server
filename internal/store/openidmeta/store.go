// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta

import (
	"sync"
)

var (
	_ Reader = (*Store)(nil)
	_ Writer = (*Store)(nil)
)

// Reader lets the consumer read entries from [Store].
type Reader interface {
	Read(key string) (Data, bool)
}

// Writer lets the consumer write entries to [Store].
type Writer interface {
	Write(key string, data Data)
	Delete(key string)
}

// Store is a thread safe in-memory store that can be used to
// read and write openid discovery metadata. Mind that the store
// does not perform any validation on the inputs.
type Store struct {
	mutex sync.RWMutex
	store map[string]Data
}

// Data holds openid discovery metadata.
type Data struct {
	Config []byte
	JWKS   []byte
}

// NewStore returns a ready for use [Store].
func NewStore() *Store {
	return &Store{
		store: make(map[string]Data),
	}
}

func copyData(data Data) Data {
	out := Data{
		Config: make([]byte, len(data.Config)),
		JWKS:   make([]byte, len(data.JWKS)),
	}
	copy(out.Config, data.Config)
	copy(out.JWKS, data.JWKS)
	return out
}

// Read retrieves an entry from the [Store].
func (s *Store) Read(key string) (Data, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.store[key]
	if ok {
		return copyData(data), ok
	}
	return Data{}, ok
}

// Write sets and entry to the [Store].
// If the entry exists it is overwritten.
func (s *Store) Write(key string, data Data) {
	d := copyData(data)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.store[key] = d
}

// Delete removes an entry from the [Store].
func (s *Store) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.store, key)
}

// Len returns the number of entries in the [Store].
func (s *Store) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.store)
}
