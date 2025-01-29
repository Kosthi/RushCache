package lru

import (
	"container/list"
)

type Value interface {
	Len() int
}

type entry struct {
	key   string
	value Value
}

type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nbytes    int64                         // 当前使用的内存
	ll        *list.List                    // 双向链表
	cache     map[string]*list.Element      // 字典
	OnEvicted func(key string, value Value) // 某条记录被移出时的回调函数
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// RemoveOldest 移除最老的一个元素
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 链表尾部是最久没有被使用的
	if ele != nil {
		c.ll.Remove(ele)         // 从双向链表中删除
		kv := ele.Value.(*entry) // 得到该元素的对应的 kv
		delete(c.cache, kv.key)  // 从 map 中删除该 key
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value) // 调用驱逐函数
		}
	}
}

// Add 添加元素
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { // 如果 key 已经在缓存中，替换 value
		c.ll.MoveToFront(ele)                                  // 最近被使用过，移动到链表头
		kv := ele.Value.(*entry)                               // 访问链表某个节点上的值 ele.Value
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // value 大小的增量
		kv.value = value                                       // 替换 value
	} else {
		v := c.ll.PushFront(&entry{key, value})
		c.cache[key] = v
		c.nbytes += int64(len(key)) + int64(value.Len()) // kv 的总长度
	}
	// 如果可用空间满了，不断移除最老的元素，直到空间空闲
	for c.maxBytes > 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get 获取元素
func (c *Cache) Get(key string) (Value, bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 最近被使用了，移动到链表头
		return ele.Value.(*entry).value, true
	}
	return nil, false
}

// Len 缓存条目数量
func (c *Cache) Len() int {
	return c.ll.Len()
}
