package collection

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "create new string queue",
		},
		{
			name: "create new int queue",
		},
		{
			name: "create new struct queue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test different types
			stringQueue := NewQueue[string]()
			if stringQueue == nil {
				t.Error("NewQueue[string]() returned nil")
			}
			if stringQueue.Len() != 0 {
				t.Errorf("New queue should be empty, got length %d", stringQueue.Len())
			}

			intQueue := NewQueue[int]()
			if intQueue == nil {
				t.Error("NewQueue[int]() returned nil")
			}
			if intQueue.Len() != 0 {
				t.Errorf("New queue should be empty, got length %d", intQueue.Len())
			}

			type testStruct struct {
				Name string
				ID   int
			}
			structQueue := NewQueue[testStruct]()
			if structQueue == nil {
				t.Error("NewQueue[testStruct]() returned nil")
			}
			if structQueue.Len() != 0 {
				t.Errorf("New queue should be empty, got length %d", structQueue.Len())
			}
		})
	}
}

func TestQueue_Push(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
		want   int
	}{
		{
			name:   "push single value",
			values: []string{"first"},
			want:   1,
		},
		{
			name:   "push multiple values",
			values: []string{"first", "second", "third"},
			want:   3,
		},
		{
			name:   "push empty string",
			values: []string{""},
			want:   1,
		},
		{
			name:   "push duplicate values",
			values: []string{"same", "same", "same"},
			want:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewQueue[string]()
			for _, v := range tt.values {
				q.Push(v)
			}

			if got := q.Len(); got != tt.want {
				t.Errorf("Queue.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueue_Pop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		values   []string
		want     []string
		popCount int
		wantLen  int
	}{
		{
			name:     "pop from single element queue",
			values:   []string{"only"},
			popCount: 1,
			want:     []string{"only"},
			wantLen:  0,
		},
		{
			name:     "pop multiple elements FIFO order",
			values:   []string{"first", "second", "third"},
			popCount: 3,
			want:     []string{"first", "second", "third"},
			wantLen:  0,
		},
		{
			name:     "pop partial elements",
			values:   []string{"a", "b", "c", "d"},
			popCount: 2,
			want:     []string{"a", "b"},
			wantLen:  2,
		},
		{
			name:     "pop from empty queue",
			values:   []string{},
			popCount: 1,
			want:     []string{""},
			wantLen:  0,
		},
		{
			name:     "pop more than available",
			values:   []string{"single"},
			popCount: 3,
			want:     []string{"single", "", ""},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewQueue[string]()
			for _, v := range tt.values {
				q.Push(v)
			}

			var results []string
			for i := 0; i < tt.popCount; i++ {
				results = append(results, q.Pop())
			}

			if len(results) != len(tt.want) {
				t.Errorf("Got %d results, want %d", len(results), len(tt.want))
			}

			for i, got := range results {
				if got != tt.want[i] {
					t.Errorf("Pop() result[%d] = %v, want %v", i, got, tt.want[i])
				}
			}

			if got := q.Len(); got != tt.wantLen {
				t.Errorf("Final queue length = %v, want %v", got, tt.wantLen)
			}
		})
	}
}

func TestQueue_Peek(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		values []string
	}{
		{
			name:   "peek single element",
			values: []string{"only"},
			want:   "only",
		},
		{
			name:   "peek first of multiple elements",
			values: []string{"first", "second", "third"},
			want:   "first",
		},
		{
			name:   "peek empty queue",
			values: []string{},
			want:   "", // zero value for string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewQueue[string]()
			for _, v := range tt.values {
				q.Push(v)
			}

			originalLen := q.Len()
			got := q.Peek()

			if got != tt.want {
				t.Errorf("Queue.Peek() = %v, want %v", got, tt.want)
			}

			// Peek should not modify the queue
			if q.Len() != originalLen {
				t.Errorf("Peek modified queue length: got %d, want %d", q.Len(), originalLen)
			}

			// Verify the element is still there (only if queue was not empty)
			if q.Len() > 0 {
				peeked := q.Pop()
				if peeked != tt.want {
					t.Errorf("After peek, pop returned %v, want %v", peeked, tt.want)
				}
			}
		})
	}
}

func TestQueue_Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pushCount int
		popCount  int
		want      int
	}{
		{
			name: "empty queue length",
			want: 0,
		},
		{
			name:      "after pushes",
			pushCount: 5,
			want:      5,
		},
		{
			name:      "after pushes and pops",
			pushCount: 5,
			popCount:  2,
			want:      3,
		},
		{
			name:      "after equal pushes and pops",
			pushCount: 3,
			popCount:  3,
			want:      0,
		},
		{
			name:      "after more pops than pushes",
			pushCount: 2,
			popCount:  5,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewQueue[int]()

			for i := 0; i < tt.pushCount; i++ {
				q.Push(i)
			}

			for i := 0; i < tt.popCount; i++ {
				q.Pop()
			}

			if got := q.Len(); got != tt.want {
				t.Errorf("Queue.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueue_Iter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		values        []int
		expectedOrder []int
		stopAfter     int
		finalLen      int
	}{
		{
			name:          "iterate empty queue",
			values:        []int{},
			expectedOrder: []int{},
			finalLen:      0,
		},
		{
			name:          "iterate single element",
			values:        []int{42},
			expectedOrder: []int{42},
			finalLen:      0,
		},
		{
			name:          "iterate multiple elements",
			values:        []int{1, 2, 3, 4, 5},
			expectedOrder: []int{1, 2, 3, 4, 5},
			finalLen:      0,
		},
		{
			name:          "early termination",
			values:        []int{1, 2, 3, 4, 5},
			stopAfter:     3,
			expectedOrder: []int{1, 2, 3},
			finalLen:      2, // Remaining elements: 4, 5
		},
		{
			name:          "stop immediately",
			values:        []int{1, 2, 3},
			stopAfter:     1,
			expectedOrder: []int{1},
			finalLen:      2, // Remaining elements: 2, 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := NewQueue[int]()
			for _, v := range tt.values {
				q.Push(v)
			}

			var results []int
			count := 0
			for v := range q.Iter {
				results = append(results, v)
				count++
				if tt.stopAfter > 0 && count >= tt.stopAfter {
					break
				}
			}

			if len(results) != len(tt.expectedOrder) {
				t.Errorf("Iter yielded %d values, expected %d", len(results), len(tt.expectedOrder))
			}

			for i, got := range results {
				if got != tt.expectedOrder[i] {
					t.Errorf("Iter result[%d] = %v, want %v", i, got, tt.expectedOrder[i])
				}
			}

			if finalLen := q.Len(); finalLen != tt.finalLen {
				t.Errorf("Queue length after iteration = %d, want %d", finalLen, tt.finalLen)
			}
		})
	}
}

func TestQueue_IterDestructive(t *testing.T) {
	t.Parallel()

	// Test that Iter is destructive (removes elements as it iterates)
	q := NewQueue[string]()
	values := []string{"a", "b", "c"}
	for _, v := range values {
		q.Push(v)
	}

	if q.Len() != 3 {
		t.Fatalf("Expected queue length 3, got %d", q.Len())
	}

	// First iteration should consume all elements
	var first []string
	for v := range q.Iter {
		first = append(first, v)
	}

	if len(first) != 3 {
		t.Errorf("First iteration got %d elements, want 3", len(first))
	}

	if q.Len() != 0 {
		t.Errorf("Queue should be empty after iteration, got length %d", q.Len())
	}

	// Second iteration should yield nothing
	var second []string
	for v := range q.Iter {
		second = append(second, v)
	}

	if len(second) != 0 {
		t.Errorf("Second iteration should yield nothing, got %d elements", len(second))
	}
}

func TestQueue_MixedOperations(t *testing.T) {
	t.Parallel()

	// Test complex scenarios mixing all operations
	q := NewQueue[int]()

	// Initially empty
	if q.Len() != 0 {
		t.Errorf("New queue should be empty")
	}

	// Push some values
	q.Push(1)
	q.Push(2)
	q.Push(3)

	if q.Len() != 3 {
		t.Errorf("After 3 pushes, length should be 3, got %d", q.Len())
	}

	// Peek should return first element without removing it
	if got := q.Peek(); got != 1 {
		t.Errorf("Peek() = %d, want 1", got)
	}

	if q.Len() != 3 {
		t.Errorf("After peek, length should still be 3, got %d", q.Len())
	}

	// Pop should return elements in FIFO order
	if got := q.Pop(); got != 1 {
		t.Errorf("First pop() = %d, want 1", got)
	}

	if q.Len() != 2 {
		t.Errorf("After one pop, length should be 2, got %d", q.Len())
	}

	// Add more elements
	q.Push(4)
	q.Push(5)

	if q.Len() != 4 {
		t.Errorf("After adding 2 more elements, length should be 4, got %d", q.Len())
	}

	// Verify order is preserved: should be [2, 3, 4, 5]
	expected := []int{2, 3, 4, 5}
	for i, want := range expected {
		if got := q.Pop(); got != want {
			t.Errorf("Pop %d: got %d, want %d", i, got, want)
		}
	}

	if q.Len() != 0 {
		t.Errorf("Queue should be empty after popping all elements, got length %d", q.Len())
	}
}

func TestQueue_TypeSafety(t *testing.T) {
	t.Parallel()

	// Test that different typed queues work independently
	stringQueue := NewQueue[string]()
	intQueue := NewQueue[int]()

	stringQueue.Push("hello")
	intQueue.Push(42)

	if stringQueue.Pop() != "hello" {
		t.Error("String queue should return string")
	}

	if intQueue.Pop() != 42 {
		t.Error("Int queue should return int")
	}

	// Test with custom types
	type Person struct {
		Name string
		Age  int
	}

	personQueue := NewQueue[Person]()
	person := Person{Name: "Alice", Age: 30}
	personQueue.Push(person)

	if got := personQueue.Pop(); got != person {
		t.Errorf("Person queue returned %v, want %v", got, person)
	}
}

// Benchmark tests
func BenchmarkQueue_Push(b *testing.B) {
	q := NewQueue[int]()

	for i := 0; b.Loop(); i++ {
		q.Push(i)
	}
}

func BenchmarkQueue_Pop(b *testing.B) {
	q := NewQueue[int]()

	// Pre-populate queue
	for i := 0; i < b.N; i++ {
		q.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pop()
	}
}

func BenchmarkQueue_PushPop(b *testing.B) {
	q := NewQueue[int]()

	for i := 0; b.Loop(); i++ {
		q.Push(i)
		q.Pop()
	}
}

func BenchmarkQueue_Iter(b *testing.B) {
	q := NewQueue[int]()

	// Pre-populate queue
	for i := 0; b.Loop(); i++ {
		q.Push(i)
	}

	b.ResetTimer()
	for range q.Iter {
		// Just iterate
	}
}
