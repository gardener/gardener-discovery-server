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

type Reader interface {
	Load(key string) (Data, bool)
}

type Writer interface {
	Set(key string, data Data)
	Delete(key string)
}

type Store struct {
	mutex sync.RWMutex
	store map[string]Data
}

type Data struct {
	Config []byte
	JWKS   []byte
}

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

func (s *Store) Load(key string) (Data, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	data, ok := s.store[key]
	if ok {
		return copyData(data), ok
	}
	return Data{}, ok
}

func (s *Store) Set(key string, data Data) {
	d := copyData(data)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.store[key] = d
}

func (s *Store) Delete(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.store, key)
}

func (s *Store) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	len := len(s.store)
	return len
}
