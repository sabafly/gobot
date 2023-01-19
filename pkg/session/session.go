/*
	Copyright (C) 2022  ikafly144

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
package session

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type sessionData[T any] struct {
	id   *string
	data *T
}

type session[T any] struct {
	data *sync.Map
}

// 新しいセッションを作成
func newSession[T any]() (s *session[T]) {
	s = &session[T]{}
	s.data = new(sync.Map)
	return
}

// idからデータを取得
func (s *session[T]) get(id string) (*sessionData[T], error) {
	m := s.data
	var d any
	var ok bool
	if d, ok = m.Load(id); !ok {
		return nil, fmt.Errorf("not found id: %v", id)
	}
	value, ok := d.(*sessionData[T])
	if !ok {
		return nil, fmt.Errorf("cannot convert: %v", id)
	}
	return value, nil
}

// セッションにデータを上書きする
func (s *session[T]) set(data *T) (id string) {
	m := s.data
	id = uuid.New().String()
	m.Delete(id)
	m.Store(id, data)
	return id
}

// idを指定してセッションにデータを上書きする
func (s *session[T]) setWithID(data *T, id string) {
	m := s.data
	m.Delete(id)
	m.Store(id, data)
}

// idのデータを削除する
func (s *session[T]) remove(id string) {
	m := s.data
	m.Delete(id)
}

// セッションデータからidを取得する
func (sd *sessionData[T]) ID() (res string) {
	return *sd.id
}

// セッションデータから値を取得する
func (sd *sessionData[T]) Data() (res T) {
	return *sd.data
}
