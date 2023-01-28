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
package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// データベース接続を管理する構造体
type DatabaseManager struct {
	db *gorm.DB
}

// データベース構造体を生成
func NewDatabase() *DatabaseManager {
	return &DatabaseManager{}
}

// データベースに接続
func (d *DatabaseManager) Connect(host, port, user, pass, name string, logLevel logger.LogLevel) (err error) {
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + name + " sslmode=disable TimeZone=Asia/Tokyo"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
	if err != nil {
		return err
	}
	d.db = db
	return nil
}

// テーブルを作成
func (d *DatabaseManager) Create(data *any) (err error) {
	result := d.db.Create(&data)
	if err := result.Error; err != nil {
		return err
	}
	return nil
}

// 渡されたデータ型に一致する最初のレコードを取得します
// 条件にプライマリキーの値を渡すとそれに一致するレコードを取得します
//
// プライマリキーがString型の場合、SQLインジェクションを回避するために条件は
// "id = ?", "217f7f2c-d648-4eed-ab09-b199a0f168f7"
// のようにプレースホルダを使用して下さい
//
// その他の条件はgormのドキュメントを確認してください
// https://gorm.io/ja_JP/docs/query.html#%E5%8F%96%E5%BE%97%E6%9D%A1%E4%BB%B6
func (d *DatabaseManager) First(v *any, cond ...any) (err error) {
	result := d.db.First(&v, cond...)
	if err := result.Error; err != nil {
		return err
	}
	return nil
}

// 渡されたデータ型に一致するすべてのレコードを取得します
// 条件にプライマリキーの値を渡すとそれに一致するレコードを取得します
//
// プライマリキーがString型の場合、SQLインジェクションを回避するために条件は
// "id = ?", "217f7f2c-d648-4eed-ab09-b199a0f168f7"
// のようにプレースホルダを使用して下さい
//
// その他の条件はgormのドキュメントを確認してください
// https://gorm.io/ja_JP/docs/query.html#%E5%8F%96%E5%BE%97%E6%9D%A1%E4%BB%B6
func (d *DatabaseManager) Find(v *any, cond ...any) (err error) {
	result := d.db.Find(&v, cond...)
	if err := result.Error; err != nil {
		return err
	}
	return nil
}

// 渡されたデータのプライマリキーに一致する行がある場合その行を更新します
// ない場合、新たに行を挿入します
func (d *DatabaseManager) Save(v *any) (err error) {
	result := d.db.Save(&v)
	if err := result.Error; err != nil {
		return err
	}
	return nil
}

// 渡されたデータに一致するレコードを削除します
func (d *DatabaseManager) Delete(v *any, cond ...any) (err error) {
	result := d.db.Delete(v, cond...)
	if err := result.Error; err != nil {
		return err
	}
	return nil
}
