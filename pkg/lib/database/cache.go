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

// type Data interface {
// }

// type CacheManagerInterface[T Data] interface {
// 	Delete(string)
// 	Set(string, T)
// 	Get(string) (*T, error)
// }

// type DatabaseManagerWithCache[T Data] struct {
// 	Cache    CacheManagerInterface[T]
// 	Database *DatabaseManager
// }

// func NewDataBaseWithCache[T Data](c CacheManagerInterface[T], db *DatabaseManager) *DatabaseManagerWithCache[T] {
// 	return &DatabaseManagerWithCache[T]{
// 		Cache:    c,
// 		Database: db,
// 	}
// }

// func (d *DatabaseManagerWithCache[T]) Set(key string, v T) error {
// 	err := d.Database.Create(&v)
// 	if err != nil {
// 		return err
// 	}
// 	d.Cache.Set(key, v)
// 	return nil
// }

// func (d *DatabaseManagerWithCache[T]) Delete(key string, v T) error {
// 	err := d.Database.Delete(&v)
// 	if err != nil {
// 		return err
// 	}
// 	d.Cache.Set(key, v)
// 	return nil
// }
