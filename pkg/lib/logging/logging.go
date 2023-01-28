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
package logging

import (
	"fmt"
	"log"
	"runtime"

	"github.com/mattn/go-colorable"
	"github.com/sabafly/gobot/pkg/lib/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	conf := zap.NewDevelopmentEncoderConfig()
	conf.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger = zap.New(
		zapcore.NewCore(zapcore.NewConsoleEncoder(conf),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zap.DebugLevel,
		),
	)
}

// ログレベルを環境変数から取得
func SetLogLevel() zapcore.Level {
	logLevel := env.LogLevel
	switch logLevel {
	case "INFO", "info":
		return zap.InfoLevel
	case "DEBUG", "debug":
		return zap.DebugLevel
	case "ERROR", "error":
		return zap.WarnLevel
	case "WARNING", "warning":
		return zap.WarnLevel
	default:
		return zap.InfoLevel
	}
}

// デバッグレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Debug(format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		caller := fmt.Sprintf("@%s:%d: ", file, line)
		logger.Debug(fmt.Sprintf(caller+format, args...))
	} else {
		logger.Debug(fmt.Sprintf(format, args...))
	}
}

// インフォレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Info(format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		caller := fmt.Sprintf("@%s:%d: ", file, line)
		logger.Info(fmt.Sprintf(caller+format, args...))
	} else {
		logger.Info(fmt.Sprintf(format, args...))
	}
}

// ワーニングレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Warning(format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		caller := fmt.Sprintf("@%s:%d: ", file, line)
		logger.Warn(fmt.Sprintf(caller+format, args...))
	} else {
		logger.Warn(fmt.Sprintf(format, args...))
	}
}

// エラーレベルのログを出力
//
// fmt.Printfに沿ったフォーマットを使用する
func Error(format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		caller := fmt.Sprintf("@%s:%d: ", file, line)
		logger.Error(fmt.Sprintf(caller+format, args...))
	} else {
		logger.Error(fmt.Sprintf(format, args...))
	}
}

// エラーレベルのログを出力してプロセスを異常終了
//
// fmt.Printfに沿ったフォーマットを使用する
func Fatal(format string, args ...any) {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		caller := fmt.Sprintf("@%s:%d: ", file, line)
		logger.Fatal(fmt.Sprintf(caller+format, args...))
	} else {
		logger.Fatal(fmt.Sprintf(format, args...))
	}
}

// ライターを返す
func Logger() *log.Logger {
	return zap.NewStdLog(logger)
}
