// Copyright © 2015 Martin Bertschler <mbertschler@gmail.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package users

// func SessionManager() {
// 	for {
// 		count := 0
// 		removed := 0
// 		toRemove := make([][]byte, 0)
// 		db.View(func(tx *bolt.Tx) error {
// 			tx.Bucket([]byte("sessions")).
// 				ForEach(func(key, val []byte) error {
// 				count++
// 				sess := new(Session)
// 				err := gob.NewDecoder(bytes.NewBuffer(val)).Decode(sess)
// 				if err != nil {
// 					log.Fatal("!!! Could not decode session in manager: ", err)
// 				}
// 				if time.Now().Sub(sess.LastCon) > 3*time.Minute && !sess.Bound {
// 					toRemove = append(toRemove, key)
// 					removed++
// 				}
// 				return nil
// 			})
// 			return nil
// 		})
// 		if removed != 0 {
// 			db.Update(func(tx *bolt.Tx) error {
// 				b := tx.Bucket([]byte("sessions"))
// 				for _, k := range toRemove {
// 					err := b.Delete(k)
// 					if err != nil {
// 						log.Println("!!! Could not delete a session:", err)
// 					}
// 				}
// 				return nil
// 			})
// 			log.Println(removed, "unbound Sessions removed,", count-removed, "connected")
// 		}
// 		time.Sleep(3*time.Minute + 3*time.Second)
// 	}
// }
