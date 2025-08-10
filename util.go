package main

import (
	"net/http"
	"sync"
)

type MuMap[K comparable, V any] struct {
	M  map[K]V
	Mu sync.RWMutex
}

func NewMuMap[K comparable, V any]() *MuMap[K, V] {
	return &MuMap[K, V]{
		M: make(map[K]V),
	}
}

func (m *MuMap[K, V]) Get(k K) (V, bool) {
	m.Mu.RLock()
	defer m.Mu.RUnlock()
	v, ok := m.M[k]
	return v, ok
}

func (m *MuMap[K, V]) Set(k K, v V) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.M[k] = v
}

func (m *MuMap[K, V]) Delete(k K) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	delete(m.M, k)
}

func flush[A any](ch chan A) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func execute(w http.ResponseWriter, name string, data any) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
	}
}
