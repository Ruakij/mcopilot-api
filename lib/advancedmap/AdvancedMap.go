package advancedmap

import (
	"sync"
	"time"
)

type KeyItemPair[K comparable, V any] struct {
	key  K
	item *Item[V]
}
type Item[V any] struct {
	Data          V
	deadline      time.Time
	deadlineTimer *time.Timer
}

// A generic key-value map that can store any type of value
type AdvancedMap[K comparable, V any] struct {
	data      map[K]Item[V]
	dataMutex sync.Mutex
	// Hooks
	getHook    func(K, Item[V])
	putHook    func(K, Item[V])
	removeHook func(K, Item[V])
	// A time limit for items in the map
	TimeLimit time.Duration
	// A maximum number of values in the map
	maxSize uint
}

// NewAdvancedMap creates a new AdvancedMap with the given time limit and max size
func NewAdvancedMap[K comparable, V any](timeLimit time.Duration, maxSize uint) *AdvancedMap[K, V] {
	return &AdvancedMap[K, V]{
		data:      make(map[K]Item[V]),
		dataMutex: sync.Mutex{},
		TimeLimit: timeLimit,
		maxSize:   maxSize,
	}
}

// SetGetHook sets the get hook function for the AdvancedMap
func (am *AdvancedMap[K, V]) SetGetHook(getHook func(K, Item[V])) {
	am.dataMutex.Lock()
	defer am.dataMutex.Unlock()
	am.getHook = getHook
}

// SetPutHook sets the put hook function for the AdvancedMap
func (am *AdvancedMap[K, V]) SetPutHook(putHook func(K, Item[V])) {
	am.dataMutex.Lock()
	defer am.dataMutex.Unlock()
	am.putHook = putHook
}

// SetRemoveHook sets the remove hook function for the AdvancedMap
func (am *AdvancedMap[K, V]) SetRemoveHook(removeHook func(K, Item[V])) {
	am.dataMutex.Lock()
	defer am.dataMutex.Unlock()
	am.removeHook = removeHook
}

func (item *Item[V]) Expired() bool {
	return item.deadline.Before(time.Now())
}

func (m *AdvancedMap[K, V]) setDeadline(key K, item *Item[V]) {
	if m.TimeLimit > 0 {
		item.deadline = time.Now().Add(m.TimeLimit)
		if item.deadlineTimer == nil {
			// Start timer
			item.deadlineTimer = time.AfterFunc(m.TimeLimit, func() {
				m.Remove(key)
			})
		} else {
			// Reset timer
			item.deadlineTimer.Reset(m.TimeLimit)
		}
	}
}

func (m *AdvancedMap[K, V]) getOldest() *KeyItemPair[K, V] {
	if len(m.data) == 0 {
		return nil
	}

	var oldest *KeyItemPair[K, V]
	for key, item := range m.data {
		if !item.deadline.IsZero() && (oldest == nil || oldest.item.deadline.Compare(item.deadline) > 0) {
			oldest = &KeyItemPair[K, V]{
				key,
				&item,
			}
		}
	}
	return oldest
}

func (m *AdvancedMap[K, V]) removeOldest() {
	keyItemPair := m.getOldest()
	if keyItemPair != nil {
		m.Remove(keyItemPair.key)
	}
}

func (m *AdvancedMap[K, V]) Get(key K) (V, bool) {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()

	item, ok := m.data[key]

	if m.getHook != nil {
		m.getHook(key, item)
	}

	// Reset deadline since it got accessed
	m.setDeadline(key, &item)

	return item.Data, ok
}

func (m *AdvancedMap[K, V]) Put(key K, value V) {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()

	item := Item[V]{Data: value}
	m.setDeadline(key, &item)
	m.data[key] = item

	if m.putHook != nil {
		m.putHook(key, item)
	}

	if len(m.data) > int(m.maxSize) {
		m.removeOldest()
	}
}

func (m *AdvancedMap[K, V]) Remove(key K) {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()

	item, ok := m.data[key]
	if !ok {
		return
	}

	if m.removeHook != nil {
		m.removeHook(key, item)
	}

	delete(m.data, key)
}

func (m *AdvancedMap[K, V]) RemoveWithoutHooks(key K) {
	m.dataMutex.Lock()
	defer m.dataMutex.Unlock()

	_, ok := m.data[key]
	if !ok {
		return
	}

	delete(m.data, key)
}
