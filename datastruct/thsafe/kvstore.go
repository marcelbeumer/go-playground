package thsafe

import "sync"

type Map[K comparable, T any] struct {
	v  map[K]T
	mu sync.RWMutex
}

func (u *Map[K, T]) Keys() []K {
	u.mu.RLock()
	defer u.mu.RUnlock()
	keys := make([]K, 0, len(u.v))
	for k := range u.v {
		keys = append(keys, k)
	}
	return keys
}

func (u *Map[K, T]) Values() []T {
	u.mu.RLock()
	defer u.mu.RUnlock()
	values := make([]T, 0, len(u.v))
	for _, v := range u.v {
		values = append(values, v)
	}
	return values
}

func (u *Map[K, T]) Map() map[K]T {
	u.mu.RLock()
	defer u.mu.RUnlock()
	m := make(map[K]T, len(u.v))
	for k, v := range u.v {
		m[k] = v
	}
	return m
}

func (u *Map[K, T]) Get(key K) (T, bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	i, ok := u.v[key]
	return i, ok
}

func (u *Map[K, T]) Set(key K, value T) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.v[key] = value
}

func (u *Map[K, T]) Delete(key K) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	var ok bool
	if _, ok = u.v[key]; ok {
		delete(u.v, key)
	}
	return ok
}

func NewMap[K comparable, T any]() *Map[K, T] {
	return &Map[K, T]{
		v:  make(map[K]T),
		mu: sync.RWMutex{},
	}
}
