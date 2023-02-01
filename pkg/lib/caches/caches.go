/*
	Copyright (C) 2022-2023  ikafly144

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package caches

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// キャッシュのIDとデータと最後にアクセスされた時を格納する
type Cache[T any] struct {
	ID   string
	Data *T
	ctx  context.Context
	del  func()
}

// キャッシュを管理する
type CacheManager[T any] struct {
	sync    sync.Mutex
	caches  map[string]Cache[T]
	timeOut *time.Duration
}

type key int

const (
	keyID key = iota
	keyParent
)

// 新たなキャッシュ管理インスタンスを生成する
func NewCacheManager[T any](timeOut *time.Duration) *CacheManager[T] {
	g := new(CacheManager[T])
	g.caches = make(map[string]Cache[T])
	g.timeOut = timeOut
	return g
}

// 新たな削除用コンテキストを作成する
func (ch *Cache[T]) newContext(parent *CacheManager[T], id string, timeout *time.Duration) (ctx context.Context, del func()) {
	if timeout != nil {
		// nilじゃなかったら
		ch.ctx, ch.del = context.WithTimeout(context.WithValue(context.WithValue(context.Background(), keyID, id), keyParent, parent), *timeout)
	} else {
		// nilだったら
		ch.ctx, ch.del = context.WithCancel(context.WithValue(context.WithValue(context.Background(), keyID, id), keyParent, parent))
	}

	go ch.closer()

	return ch.ctx, ch.del
}

// クローズするやつ
// クローズするときは必ずLockしてから
func (ch *Cache[T]) closer() {
	<-ch.ctx.Done()
	key := ch.ctx.Value(keyID).(string)
	parent := ch.ctx.Value(keyParent).(*CacheManager[T])
	delete(parent.caches, key)
}

// 指定されたキーで保存します
func (c *CacheManager[T]) Set(key string, v T) {
	c.sync.Lock()
	defer c.sync.Unlock()

	cache := Cache[T]{ID: key, Data: &v}
	cache.ctx, cache.del = cache.newContext(c, key, c.timeOut)

	c.caches[key] = cache
}

// データをUUIDを生成して保存します
func (c *CacheManager[T]) SetWithUUID(v T) (key string) {
	c.sync.Lock()
	defer c.sync.Unlock()

	id := uuid.New().String()
	cache := Cache[T]{ID: id, Data: &v}
	cache.ctx, cache.del = cache.newContext(c, id, c.timeOut)

	c.caches[id] = cache
	return id
}

// 指定されたキーの値を読み込みます
func (c *CacheManager[T]) Load(key string) (*T, error) {
	c.sync.Lock()
	defer c.sync.Unlock()

	v, ok := c.caches[key]
	if !ok {
		return nil, fmt.Errorf("not found in %s", key)
	}
	return v.Data, nil
}

// キャッシュを削除します
func (c *CacheManager[T]) Delete(key string) {
	c.sync.Lock()
	defer c.sync.Unlock()

	c.caches[key].del()
}

// for k, v := range cache { f(k, v) } と同義
// TODO: 非推奨
func (c *CacheManager[T]) Range(f func(string, T)) {
	c.sync.Lock()
	defer c.sync.Unlock()

	for k, c2 := range c.caches {
		f(k, *c2.Data)
	}
}

// 保存されているキャッシュの数を返す
func (c *CacheManager[T]) Len() int {
	c.sync.Lock()
	defer c.sync.Unlock()
	return len(c.caches)
}
