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
package botlib

// TODO: 独自のレートリミットを作る

// import (
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"sync"
// 	"sync/atomic"
// 	"time"
// )

// type RateLimiter struct {
// 	sync.Mutex
// 	buckets   map[string]*Bucket
// 	Remaining int
// }

// func NewRateLimiter() *RateLimiter {
// 	return &RateLimiter{
// 		buckets: make(map[string]*Bucket),
// 	}
// }

// type Bucket struct {
// 	sync.Mutex
// 	Remaining int
// 	limit     int
// 	reset     time.Time
// 	global    *int64

// 	lastReset time.Time
// 	UserData  any
// }

// func (b *Bucket) Release(headers http.Header) (err error) {
// 	defer b.Unlock()

// 	if headers == nil {
// 		return nil
// 	}

// 	remaining := headers.Get("X-RateLimit-Remaining")
// 	reset := headers.Get("X-RateLimit-Reset")
// 	global := headers.Get("X-RateLimit-Global")
// 	resetAfter := headers.Get("X-RateLimit-Reset-After")

// 	// TODO:理解する

// 	// Update global and per bucket reset time if the proper headers are available
// 	// If global is set, then it will block all buckets until after Retry-After
// 	// If Retry-After without global is provided it will use that for the new reset
// 	// time since it's more accurate than X-RateLimit-Reset.
// 	// If Retry-After after is not proided, it will update the reset time from X-RateLimit-Reset
// 	if resetAfter != "" {
// 		parsedAfter, err := strconv.ParseFloat(resetAfter, 64)
// 		if err != nil {
// 			return err
// 		}

// 		whole, frac := math.Modf(parsedAfter)
// 		resetAt := time.Now().Add(time.Duration(whole) * time.Second).Add(time.Duration(frac*1000) * time.Millisecond)

// 		// Lock either this single bucket or all buckets
// 		if global != "" {
// 			atomic.StoreInt64(b.global, resetAt.UnixNano())
// 		} else {
// 			b.reset = resetAt
// 		}
// 	} else if reset != "" {
// 		// Calculate the reset time by using the date header returned from discord
// 		discordTime, err := http.ParseTime(headers.Get("Date"))
// 		if err != nil {
// 			return err
// 		}

// 		unix, err := strconv.ParseFloat(reset, 64)
// 		if err != nil {
// 			return err
// 		}

// 		// Calculate the time until reset and add it to the current local time
// 		// some extra time is added because without it i still encountered 429's.
// 		// The added amount is the lowest amount that gave no 429's
// 		// in 1k requests
// 		whole, frac := math.Modf(unix)
// 		delta := time.Unix(int64(whole), 0).Add(time.Duration(frac*1000)*time.Millisecond).Sub(discordTime) + time.Millisecond*250
// 		b.reset = time.Now().Add(delta)
// 	}

// 	// Udpate remaining if header is present
// 	if remaining != "" {
// 		parsedRemaining, err := strconv.ParseInt(remaining, 10, 32)
// 		if err != nil {
// 			return err
// 		}
// 		b.Remaining = int(parsedRemaining)
// 	}

// 	return nil
// 	return nil
// }
