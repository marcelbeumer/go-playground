package thsafe

import "sync"

type stackNode[T any] struct {
	value T
	back  *stackNode[T]
}

type Stack[T any] struct {
	len int
	top *stackNode[T]
	mu  sync.RWMutex
}

func (s *Stack[T]) Push(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.len++
	s.top = &stackNode[T]{
		value: v,
		back:  s.top,
	}
}

func (s *Stack[T]) Pop() (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.top != nil {
		s.len--
		v := s.top
		s.top = v.back
		return v.value, true
	}
	return *new(T), false
}

func (s *Stack[T]) Top() (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.top != nil {
		return s.top.value, true
	}
	return *new(T), false
}

func (s *Stack[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.len
}
