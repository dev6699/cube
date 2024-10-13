package queue

type Queue[T any] struct {
	start  *node[T]
	end    *node[T]
	length int
}

type node[T any] struct {
	value T
	next  *node[T]
}

// Create a new queue
func New[T any]() *Queue[T] {
	return &Queue[T]{
		start:  nil,
		end:    nil,
		length: 0,
	}
}

// Dequeue takes the next item off the front of the queue
func (q *Queue[T]) Dequeue() (T, bool) {
	var t T
	if q.length == 0 {
		return t, false
	}

	n := q.start
	if q.length == 1 {
		q.start = nil
		q.end = nil
	} else {
		q.start = q.start.next
	}

	q.length--
	return n.value, true
}

// Enqueue puts an item on the end of a queue
func (q *Queue[T]) Enqueue(value T) {
	n := &node[T]{
		value: value,
		next:  nil,
	}

	if q.length == 0 {
		q.start = n
		q.end = n
	} else {
		q.end.next = n
		q.end = n
	}

	q.length++
}

// Len returns the number of items in the queue
func (q *Queue[T]) Len() int {
	return q.length
}

// Peek returns the first item in the queue without removing it
func (q *Queue[T]) Peek() interface{} {
	if q.length == 0 {
		return nil
	}
	return q.start.value
}
