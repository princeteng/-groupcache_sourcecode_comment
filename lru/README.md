# 1.LRU简介
LRU(Least Recently Used)是一种缓存淘汰策略.
受内存大小的限制,不能将所有的数据都缓存在内存中,当缓存超过规定的容量时,再往里面加数据就要考虑将谁先换出去,即淘汰掉.
**LRU的做法是:淘汰最近最少使用的数据.**
LRU可以通过**哈希表+双向链表**实现,**双向链表的每个结点中存储{key,value},哈希表中存储{key,key所在的结点}.**

具体实现可参考:[leetcode146——LRU 缓存机制](https://blog.csdn.net/princeteng/article/details/104499867)
[内存结构示意图](https://leetcode-cn.com/problems/lru-cache/solution/ha-xi-biao-shuang-xiang-lian-biao-java-by-liweiw-2/)

LRU最主要的两个操作为`get`和`put`(或`add`),其中`get`从缓存中获取`key`对应的`value`,`put/add`将新的数据加入缓存当中,如果加入过程中超出缓存的容量,将会导致键的淘汰.

## get操作
```bash
if key不存在:
	直接返回-1
else 
	在原链表中删除(key,value)
	将(key,value)重新放回链表的头部
	更新哈希表
	返回value
```

## put操作
```bash
if key存在：
	在原链表中删除(key, value)
	将(key,value)重新放回链表的头部
	更新哈希表
else
	if 链表长度达到上限：
		获取链表尾部的key
		在哈希表中删除key
		删除链表尾部元素
		将新的(key,value)插入链表头部
		将(key, key在链表中的位置)放入哈希表
	else
		将新的(key,value)插入链表头部
		将(key, key在链表中的位置)放入哈希表
```

# 2. groupcache中的LRU
groupcache中的LRU也是通过哈希表加双向链表实现的,具体实现在[lru](https://github.com/golang/groupcache/tree/master/lru)目录下.

## 2.1 cache结构

```go
import "container/list" //双向链表

type Cache struct {
	MaxEntries int // MaxEntries表示链表最多能容纳的结点数量,如果该字段为0说明容量没有限制
	OnEvicted func(key Key, value interface{}) // 当缓存中的一个结点被删除时,调用该函数

	ll    *list.List //双向链表
	cache map[interface{}]*list.Element // 哈希表
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{} // key必须是可以比较的,因为要在map中当key

type entry struct { //链表中每个结点的结构,key和value都是interface
	key   Key
	value interface{}
}
```

Cache即为LRU缓存,struct中包含了双向链表和哈希表,以及最大容量和结点被删除时调用的函数.
如果函数在初始化时没有指定,则不调用.

## 2.2 创建Cache New

```go
func New(maxEntries int) *Cache { // 创建一个Cache
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}
```
返回值为指针类型.

## 2.3 get操作
函数原形:`func (c *Cache) Get(key Key) (value interface{}, ok bool)`
参数:`key`表示要查找的键
返回值:`value`为`key`对应的值,`ok`表示是否命中,`true`表示命中.
```go
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
```

## 2.4 add操作
函数原形: `func (c *Cache) Add(key Key, value interface{})`
参数: 新加入的`key`和`value`
返回值: 无

```go
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
```
重点要理清add操作的逻辑流程.

## 2.5 RemoveOldest 删除操作
删除最近最少使用的`item`的操作由函数`RemoveOldest`完成.
`RemoveOldest`首先获取最近最少使用的结点,然后调用函数`removeElement`将其删除.

删除过程包括: 从链表中删除对应结点, 从哈希表中删除对应项.

```go
// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() { // 删除最近最久没有使用的
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
```

## 2.6 清空cache

```go
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
```


# 3. 总结
lru是groupcache中基础且简单的内容,如果了解lru算法,实现一个lru并不是特别难,但通过阅读源码,可以巩固语法.