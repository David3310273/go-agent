package utils

type TaskManager[T any] struct {
	limiter     chan struct{}
	tasks       []func() (T, error)
	concurrency int
}

func (t *TaskManager[T]) Create(limit int, tasks []func() (T, error)) *TaskManager[T] {
	t.limiter = make(chan struct{}, limit)
	t.concurrency = limit
	t.tasks = tasks
	return t
}

func (t *TaskManager[T]) Await(task func() (T, error)) (T, error) {
	waitChan := make(chan struct{})
	var result T
	var err error
	go func() {
		result, err = task()
		waitChan <- struct{}{}
	}()
	<-waitChan

	return result, err
}

// func (t *TaskManager[T]) Parallel(task []func() (T, error)) ([]T, []error) {

// }
