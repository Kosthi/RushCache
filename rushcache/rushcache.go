package rushcache

import (
	"fmt"
	"log"
	pb "rushcache/rushcachepb"
	"rushcache/singleflight"
	"sync"
)

// Getter 接口
type Getter interface {
	Get(key string) ([]byte, error) // 回调函数
}

// GetterFunc 函数类型
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache Cache
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("getter is nil")
	}
	mu.RLock()
	defer mu.RUnlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: Cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	if g, ok := groups[name]; ok {
		return g
	}
	return nil
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// getLocally 调用用户回调函数从数据源获取数据，并且将数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 从数据源（数据库，文件等）获取数据，回调函数由用户提供
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) load(key string) (value ByteView, err error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err != nil {
					return value, err
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (value ByteView, err error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	resp := &pb.Response{}
	err = peer.Get(req, resp)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: cloneBytes(resp.Value)}, nil
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is empty")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[RushCache] hit from cache", key)
		return v, nil
	}
	return g.load(key)
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
