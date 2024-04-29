package async

import "sync"

// Map is a generic wrapper around concurrent sync.Map.
type Map[K comparable, V any] interface {
	// CompareAndDelete deletes the entry for key if its value is equal to old.
	// If there is no current value for key in the map, CompareAndDelete returns false
	// (even if the old value is the nil interface value).
	CompareAndDelete(key K, old V) (deleted bool)

	// CompareAndSwap swaps the old and new values for key if the value stored
	// in the map is equal to old.The old value must be of a comparable type.
	CompareAndSwap(key K, old, new V) bool

	// Delete deletes the value for a key.
	Delete(key K)

	// Len iterates the map, counts the number of keys, and returns the result.
	Len() int

	// Load returns the value stored in the map for a key, or nil if no value is present.
	// The ok result indicates whether value was found in the map.
	Load(key K) (value V, ok bool)

	// LoadAndDelete deletes the value for a key, returning the previous value if any.
	// The loaded result reports whether the key was present.
	LoadAndDelete(key K) (value V, loaded bool)

	// LoadOrStore returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value. The loaded result is true
	// if the value was loaded, false if stored.
	LoadOrStore(key K, value V) (actual V, loaded bool)

	// Range calls f sequentially for each key and value present in the map.
	// If f returns false, range stops the iteration.
	Range(f func(key K, value V) bool)

	// Store sets the value for a key.
	Store(key K, value V)

	// Swap swaps the value for a key and returns the previous value if any.
	// The loaded result reports whether the key was present.
	Swap(key K, value V) (previous V, loaded bool)
}

// NewMap returns a new concurrent map.
func NewMap[K comparable, V any]() Map[K, V] {
	return newMap[K, V]()
}

// internal

type asyncMap[K comparable, V any] struct {
	raw sync.Map
}

func newMap[K comparable, V any]() *asyncMap[K, V] {
	return &asyncMap[K, V]{}
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
// If there is no current value for key in the map, CompareAndDelete returns false
// (even if the old value is the nil interface value).
func (m *asyncMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return m.raw.CompareAndDelete(key, old)
}

// CompareAndSwap swaps the old and new values for key if the value stored
// in the map is equal to old.The old value must be of a comparable type.
func (m *asyncMap[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.raw.CompareAndSwap(key, old, new)
}

// Delete deletes the value for a key.
func (m *asyncMap[K, V]) Delete(key K) {
	m.raw.Delete(key)
}

// Len iterates the map, counts the number of keys, and returns the result.
func (m *asyncMap[K, V]) Len() int {
	var n int
	m.raw.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *asyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.raw.Load(key)
	if !ok {
		return value, false
	}
	return v.(V), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *asyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.raw.LoadAndDelete(key)
	return v.(V), true
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value. The loaded result is true
// if the value was loaded, false if stored.
func (m *asyncMap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, loaded := m.raw.LoadOrStore(key, value)
	return v.(V), false
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (m *asyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.raw.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

// Store sets the value for a key.
func (m *asyncMap[K, V]) Store(key K, value V) {
	m.raw.Store(key, value)
}

// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (m *asyncMap[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	v, loaded := m.raw.Swap(key, value)
	return v.(V), true
}
