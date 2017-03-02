package main

// LRU represents an LRU (or queue) over Items. We specialize to Item's as
// Golang lacks a reasonable generics / polymorphic type system to handle this
// well.
type LRU struct {
	head *Item
	tail *Item
}

// LRUElem represents an element in the LRU queue.
type LRUElem struct {
	next *Item
	prev *Item
}

// Erase removes the specified item from the LRU queue.
func (lru *LRU) Erase(item *Item) {
	ix := &item.lru
	if ix.prev == nil && ix.next == nil {
		// only element in lru
		lru.head = nil
		lru.tail = nil
	} else if ix.prev == nil {
		// end of lru
		ix.next.lru.prev = nil
		lru.tail = ix.next
	} else if ix.next == nil {
		// front of lru
		ix.prev.lru.next = nil
		lru.head = ix.prev
	} else {
		// middle of lru
		ix.prev.lru.next = ix.next
		ix.next.lru.prev = ix.prev
	}
	ix.next = nil
	ix.prev = nil
}

// PushBack pushes the specified item onto the end of the LRU queue.
func (lru *LRU) PushBack(item *Item) {
	item.lru.prev = nil
	item.lru.next = lru.tail
	if lru.tail != nil {
		lru.tail.lru.prev = item
	}
	lru.tail = item
	if lru.head == nil {
		lru.head = item
	}
}

// PopFront removes the first item (if one exists) from the LRU queue.
func (lru *LRU) PopFront() *Item {
	if lru.head == nil {
		return nil
	}
	item := lru.head
	lru.Erase(item)
	return item
}
