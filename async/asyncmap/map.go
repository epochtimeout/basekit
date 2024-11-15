// Copyright 2024 Ivan Korobkov. All rights reserved.
// Use of this software is governed by the MIT License
// that can be found in the LICENSE file.

package asyncmap

// Map is an abstract interface for a concurrent map.
type Map[K comparable, V any] interface {
	// Len returns the number of keys.
	Len() int

	// Clear deletes all items.
	Clear()

	// Contains returns true if a key exists.
	Contains(key K) bool

	// Get returns a key value, or false.
	Get(key K) (V, bool)

	// GetOrSet returns a key value and true, or sets a value and false.
	GetOrSet(key K, value V) (V, bool)

	// Delete deletes a key value, and returns the previous value.
	Delete(key K) (V, bool)

	// Set sets a value for a key.
	Set(key K, value V)

	// SetAbsent sets a key value if absent, returns true if set.
	SetAbsent(key K, value V) bool

	// Swap swaps a key value and returns the previous value.
	Swap(key K, value V) (V, bool)

	// Range iterates over all key-value pairs.
	// The iteration stops if the function returns false.
	Range(fn func(K, V) bool)
}
