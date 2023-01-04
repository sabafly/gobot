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

	"github.com/google/uuid"
)

func newSession[T any]() (s *session[T]) {
	s = &session[T]{}
	s.sessionData = map[string]*sessionData[T]{}
	return
}

type session[T any] struct {
	sessionData map[string]*sessionData[T]
}

func (s *session[T]) get(id string) (*sessionData[T], error) {
	if d, ok := s.sessionData[id]; ok {
		return d, nil
	}
	return &sessionData[T]{}, fmt.Errorf("not found id: %v", id)
}

func (s *session[T]) add(data *T) (id string) {
	id = uuid.New().String()
	delete(s.sessionData, id)
	s.sessionData[id] = &sessionData[T]{id: &id, data: data}
	return id
}

func (s *session[T]) addWithID(data *T, id string) {
	delete(s.sessionData, id)
	s.sessionData[id] = &sessionData[T]{id: &id, data: data}
}

func (s *session[T]) remove(id string) {
	delete(s.sessionData, id)
}

type sessionData[T any] struct {
	id   *string
	data *T
}

func (sd *sessionData[T]) ID() (res string) {
	return *sd.id
}

func (sd *sessionData[T]) Data() (res T) {
	return *sd.data
}
