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
	"errors"

	"github.com/google/uuid"
)

func New[T any]() (s *Session[T]) {
	s = &Session[T]{}
	s.sessionData = map[string]SessionData[T]{}
	return
}

type Session[T any] struct {
	sessionData map[string]SessionData[T]
}

func (s *Session[T]) Get(id string) (SessionData[T], error) {
	if d, ok := s.sessionData[id]; ok {
		return d, nil
	}
	return SessionData[T]{}, errors.New("not found")
}

func (s *Session[T]) Add(data T) (id string) {
	id = uuid.New().String()
	s.sessionData[id] = SessionData[T]{id: id, data: data}
	return id
}

func (s *Session[T]) Remove(id string) {
	delete(s.sessionData, id)
}

type SessionData[T any] struct {
	id   string
	data T
}

func (sd *SessionData[T]) ID() (res string) {
	return sd.id
}

func (sd *SessionData[T]) Data() (res T) {
	return sd.data
}
