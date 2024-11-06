/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package errors

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/internal/uuidv7"
	"log/slog"
	"runtime"
	"runtime/debug"

	"github.com/disgoorg/disgo/rest"
)

type Error interface {
	error
	File() string
	Stack() string
	ID() uuid.UUID
}

type errorImpl struct {
	err   error
	file  string
	stack string
	id    uuid.UUID
}

var _ Error = (*errorImpl)(nil)

func (e errorImpl) Error() string { return e.err.Error() }
func (e errorImpl) File() string  { return e.file }
func (e errorImpl) Stack() string { return e.stack }
func (e errorImpl) ID() uuid.UUID { return e.id }

func NewError(err error) Error {
	if err == nil {
		return nil
	}
	return newError(err, 2)
}

func newError(err error, skip int) *errorImpl {
	var ei *errorImpl
	if errors.As(err, &ei) {
		return ei
	}
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(skip, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	// TODO: なんかこうトラックIDとか言っていい感じに管理したい…

	// トラッキング
	id := uuidv7.New()

	slog.Error("エラーが生成されました", "err", err, "file", fmt.Sprintf("%s:%d", file, line), "filename", f.Name())
	e := Unwrap(err)
	if e == nil {
		e = err
	}
	var restErr rest.Error
	if errors.As(e, &restErr) {
		slog.Error("request info", "err", fmt.Errorf("%w\nurl: %s\nrq: %s\nrs: %s\nhd: %v", restErr, restErr.Request.URL, string(restErr.RqBody), string(restErr.RsBody), restErr.Response.Header))
	}
	return &errorImpl{
		err:   err,
		file:  fmt.Sprintf("%s:%d %s\n", file, line, f.Name()),
		stack: string(debug.Stack()),
		id:    id,
	}
}

type ErrorWithMessage interface {
	Key() string
}

type errorWithMessageImpl struct {
	*errorImpl
	key string
}

func (e errorWithMessageImpl) Key() string { return e.key }

func NewErrorWithMessage(err error, key string) Error {
	if err == nil {
		return nil
	}
	return &errorWithMessageImpl{
		errorImpl: newError(err, 3),
		key:       key,
	}
}
