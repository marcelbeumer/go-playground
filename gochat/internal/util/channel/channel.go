package channel

func FnToChan[T any](f func() T) <-chan T {
	done := make(chan T)
	go func() {
		defer close(done)
		done <- f()
	}()
	return done
}
