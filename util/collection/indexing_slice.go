package collection

import "sync"

type IndexingSliceItem interface {
	GetKey() string
}

// IndexingSlice 将数据存在slice中, 将数据中的key存在map里方便索引
// 由于 GetKey 不能返回泛型, 暂时不方便将 key 定义为 comparable
type IndexingSlice[T IndexingSliceItem] struct {
	sync.RWMutex
	items []T
	index map[string]int
}

func NewIndexingSlice[T IndexingSliceItem]() *IndexingSlice[T] {
	return &IndexingSlice[T]{
		index: make(map[string]int),
	}
}

func (m *IndexingSlice[T]) Len() int {
	return len(m.items)
}

func (m *IndexingSlice[T]) LenKeys() int {
	return len(m.index)
}

func (m *IndexingSlice[T]) GetByIndex(idx int) (T, bool) {
	if idx >= len(m.items) {
		var zero T
		return zero, false
	}
	return m.items[idx], true
}

func (m *IndexingSlice[T]) GetByKey(key string) (T, bool) {
	m.RLock()
	defer m.RUnlock()
	idx, ok := m.index[key]
	if !ok {
		var zero T
		return zero, false
	}
	return m.items[idx], true
}

func (m *IndexingSlice[T]) Add(value T) bool {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.index[value.GetKey()]; ok {
		return false
	}

	m.items = append(m.items, value)
	m.index[value.GetKey()] = len(m.items) - 1
	return true
}

func (m *IndexingSlice[T]) Delete(key string) {
	m.Lock()
	defer m.Unlock()

	idx, ok := m.index[key]
	if !ok {
		return
	}
	delete(m.index, key)
	// 将最后一位和需要删除的位置对调
	maxIdx := len(m.items) - 1
	if idx < maxIdx {
		last := m.items[maxIdx]
		m.items[idx] = last
		m.index[last.GetKey()] = idx
	}
	m.items = m.items[0:maxIdx]
}
