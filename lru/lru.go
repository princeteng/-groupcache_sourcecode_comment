/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lru implements an LRU cache.
package lru

import "container/list" //双向链表

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int // MaxEntries表示链表最多能容纳的结点数量,如果该字段为0说明容量没有限制

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key Key, value interface{}) // 当缓存中的一个结点被删除时,调用该函数

	ll    *list.List //双向链表
	cache map[interface{}]*list.Element // 哈希表
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

type entry struct { //链表中每个结点的结构,key和value都是interface
	key   Key
	value interface{}
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New(maxEntries int) *Cache { // 创建一个Cache
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key Key, value interface{}) { // 添加{key,value}到缓存中
	if c.cache == nil { // 如果之前缓存为空, 则先创建哈希表和链表
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok { // 如果key原来就有, 我们只需要更新key对应的值即可
		c.ll.MoveToFront(ee) // 将key对应的结点移动至链表头部, 因为刚刚被访问过
		ee.Value.(*entry).value = value // 更新key对应的value, 然后返回
		return
	}
	ele := c.ll.PushFront(&entry{key, value}) // 如果key不存在, 则创建一个结点并将其放在链表头部
	c.cache[key] = ele // 更新哈希表
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries { // 如果插入新结点后超过最大容量, 则淘汰一个键
		c.RemoveOldest() // 用于淘汰最近最少使用的
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil { // 缓存为空直接返回
		return
	}
	if ele, hit := c.cache[key]; hit { // ele表示key在链表中对应的结点,hit表示是否命中
		c.ll.MoveToFront(ele) //命中,说明缓存中有key,则将key对应的结点移动到链表头部(因为刚刚被访问)
		return ele.Value.(*entry).value, true // 返回key对应的值, true表示命中
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) { // 在cache中删除key
	if c.cache == nil { // cache为空直接返回
		return
	}
	if ele, hit := c.cache[key]; hit { // 命中之后调用removeElement将key从哈希表中删除,将key对应的结点从链表中删除
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() { // 删除最近最少使用的
	if c.cache == nil {
		return
	}
	ele := c.ll.Back() // 链表尾部的结点就是最近最少使用的
	if ele != nil {
		c.removeElement(ele) // 调用removeElement函数删除item
	}
}

func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e) // 从链表中删除key对应的结点
	kv := e.Value.(*entry)
	delete(c.cache, kv.key) // 从哈希表中删除key
	if c.OnEvicted != nil { // 如果OnEvicted被设置过, 则在删除item时要调用一个这个函数
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int { // 返回当前cache中的结点数量
	if c.cache == nil {
		return 0
	}
	return c.ll.Len() // 就是链表的长度
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() { // 清空cache, 删除所有的结点
	if c.OnEvicted != nil { // 如果OnEvicted被设置, 则删除时需要调用一个这个函数
		for _, e := range c.cache {
			kv := e.Value.(*entry)
			c.OnEvicted(kv.key, kv.value)
		}
	}
	c.ll = nil // 然后将链表清空
	c.cache = nil // 将哈希表清空
}
