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
package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/ikafly144/gobot/pkg/lib/env"
)

const (
	ERROR = iota
	WARNING
	INFO
	DEBUG
)

// ログレベルを環境変数から取得
func SetLogLevel() int {
	logLevel := env.LogLevel
	switch logLevel {
	case "INFO", "info":
		return INFO
	case "DEBUG", "debug":
		return DEBUG
	case "ERROR", "error":
		return ERROR
	case "WARNING", "warning":
		return WARNING
	default:
		return INFO
	}
}

// ログレベルを持つLogger
type BuiltinLogger struct {
	logger *log.Logger
	level  int
}

// デバッグレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Debug(format string, args ...any) {
	l := NewBuiltinLogger()
	if l.level >= DEBUG {
		prefix := fmt.Sprintf("[%s] ", "DEBG")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Printf(caller+format, args...)
		} else {
			l.logger.Printf(format, args...)
		}
	}
}

// インフォレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Info(format string, args ...any) {
	l := NewBuiltinLogger()
	if l.level >= INFO {
		prefix := fmt.Sprintf("[%s] ", "INFO")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)
		l.logger.Printf(format, args...)
	}
}

// ワーニングレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Warning(format string, args ...any) {
	l := NewBuiltinLogger()
	if l.level >= WARNING {
		prefix := fmt.Sprintf("[%s] ", "WARN")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)
		l.logger.Printf(format, args...)
	}
}

// エラーレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Error(format string, args ...any) {
	l := NewBuiltinLogger()
	if l.level >= ERROR {
		prefix := fmt.Sprintf("[%s] ", "EROR")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Printf(caller+format, args...)
		} else {
			l.logger.Printf(format, args...)
		}
	}
}

// エラーレベルのログを出力してプロセスを異常終了
//
// fmt.Printfに沿ったフォーマットを使用する
func Fatal(format string, args ...any) {
	l := NewBuiltinLogger()
	if l.level >= ERROR {
		prefix := fmt.Sprintf("[%s] ", "EROR")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Fatalf(caller+format, args...)
		} else {
			l.logger.Fatalf(format, args...)
		}
	}
}

// 新たなLoggerインスタンスを生成
func NewBuiltinLogger() *BuiltinLogger {
	return &BuiltinLogger{
		logger: log.Default(),
		level:  SetLogLevel(),
	}
}

// デバッグレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func (l *BuiltinLogger) Debug(format string, args ...any) {
	if l.level >= DEBUG {
		prefix := fmt.Sprintf("[%s] ", "DEBG")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Printf(caller+format, args...)
		} else {
			l.logger.Printf(format, args...)
		}
	}
}

// インフォレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func (l *BuiltinLogger) Info(format string, args ...any) {
	if l.level >= INFO {
		prefix := fmt.Sprintf("[%s] ", "INFO")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)
		l.logger.Printf(format, args...)
	}
}

// ワーニングレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func (l *BuiltinLogger) Warning(format string, args ...any) {
	if l.level >= WARNING {
		prefix := fmt.Sprintf("[%s] ", "WARN")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)
		l.logger.Printf(format, args...)
	}
}

// エラーレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func (l *BuiltinLogger) Error(format string, args ...any) {
	if l.level >= ERROR {
		prefix := fmt.Sprintf("[%s] ", "EROR")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Printf(caller+format, args...)
		} else {
			l.logger.Printf(format, args...)
		}
	}
}

// エラーレベルのログを出力してプロセスを異常終了
//
// fmt.Printfに沿ったフォーマットを使用する
func (l *BuiltinLogger) Fatal(format string, args ...any) {
	if l.level >= ERROR {
		prefix := fmt.Sprintf("[%s] ", "EROR")
		l.logger.SetOutput(os.Stdout)
		l.logger.SetPrefix(prefix)
		l.logger.SetFlags(log.Ldate | log.Ltime)

		_, file, line, ok := runtime.Caller(1)
		if ok {
			caller := fmt.Sprintf("@%s:%d: ", file, line)
			l.logger.Fatalf(caller+format, args...)
		} else {
			l.logger.Fatalf(format, args...)
		}
	}
}
