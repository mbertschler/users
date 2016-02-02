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

package crowd

import (
	"log"
	"sync"
	"sync/atomic"
)

// enable debug messages when store functions are called
const storeDebug = false

// memoryStore is a thread safe memory backend for the Store type. It
// implements the Storer interface and provides user and session storage.
// Do not use this directly, instead call NewMemoryStore().
// memoryStore saves the actual values behind the passed pointers.
type memoryStore struct {
	sessions      map[string]StoredSession
	sessionsMutex sync.RWMutex
	users         map[uint64]StoredUser
	usersMutex    sync.RWMutex
	userIDs       map[string]uint64
	maxUserID     uint64
}

// NewMemoryStore returns a Store with a memory backend.
func NewMemoryStore() *Store {
	var s = memoryStore{
		sessions: make(map[string]StoredSession),
		users:    make(map[uint64]StoredUser),
		userIDs:  make(map[string]uint64),
	}
	return NewStore(&s)
}

func (s *memoryStore) nextUserID() uint64 {
	return atomic.AddUint64(&s.maxUserID, 1)
}

// CountUsers returns the number of saved users
func (s *memoryStore) CountUsers() int {
	if storeDebug {
		log.Println("CountUsers")
	}
	s.usersMutex.RLock()
	count := len(s.users)
	s.usersMutex.RUnlock()
	return count
}

// GetSession gets a Session object from the memoryStore
func (s *memoryStore) GetSession(id string) (*StoredSession, error) {
	if storeDebug {
		log.Println("GetSession:", id)
	}
	s.sessionsMutex.RLock()
	sess, ok := s.sessions[id]
	s.sessionsMutex.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	return &sess, nil
}

// PutSession puts a Session object in the memoryStore
func (s *memoryStore) PutSession(sess *StoredSession) error {
	if storeDebug {
		log.Println("PutSession:", sess.ID)
	}
	s.sessionsMutex.Lock()
	s.sessions[sess.ID] = *sess
	s.sessionsMutex.Unlock()
	return nil
}

// DeleteSession deletes a session object from the memoryStore
func (s *memoryStore) DeleteSession(id string) error {
	if storeDebug {
		log.Println("DeleteSession:", id)
	}
	s.sessionsMutex.Lock()
	delete(s.sessions, id)
	s.sessionsMutex.Unlock()
	return nil
}

// ForEachSession ranges over all sessions from the memoryStore
func (s *memoryStore) ForEachSession(fn func(s *StoredSession) (del bool)) error {
	if storeDebug {
		log.Println("ForEachSession")
	}
	s.sessionsMutex.RLock()
	for k, v := range s.sessions {
		if fn(&v) {
			s.sessionsMutex.RUnlock()
			s.sessionsMutex.Lock()
			delete(s.sessions, k)
			s.sessionsMutex.Unlock()
			s.sessionsMutex.RLock()
		}
	}
	s.sessionsMutex.RUnlock()
	return nil
}

// GetUser gets a User object via the user ID from the memoryStore
func (s *memoryStore) GetUser(id uint64) (*StoredUser, error) {
	if storeDebug {
		log.Println("GetUser:", id)
	}
	s.usersMutex.RLock()
	u, ok := s.users[id]
	s.usersMutex.RUnlock()
	if !ok {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// GetUserID gets the user ID via the username from the memoryStore
func (s *memoryStore) GetUserID(username string) (uint64, error) {
	if storeDebug {
		log.Println("GetUserID:", username)
	}
	s.usersMutex.RLock()
	uid, ok := s.userIDs[username]
	s.usersMutex.RUnlock()
	if !ok {
		return 0, ErrUserNotFound
	}
	return uid, nil
}

// PutUser puts a User object in the memoryStore
func (s *memoryStore) PutUser(u *StoredUser) error {
	if storeDebug {
		log.Println("PutUser:", u.ID, u.Name)
	}
	s.usersMutex.Lock()
	s.users[u.ID] = *u
	s.usersMutex.Unlock()
	return nil
}

// AddUser puts a new User object in the memoryStore and returns the user ID
func (s *memoryStore) AddUser(u *StoredUser) (uint64, error) {
	if storeDebug {
		log.Println("AddUser:", u.ID, u.Name)
	}
	if u == nil {
		panic("AddUser: argument stored user is nil")
	}
	u.ID = s.nextUserID()
	s.usersMutex.Lock()
	s.users[u.ID] = *u
	s.userIDs[u.Name] = u.ID
	s.usersMutex.Unlock()
	return u.ID, nil
}

// RenameUser renames a user while keeping the ID the same
func (s *memoryStore) RenameUser(id uint64, newname string) error {
	if storeDebug {
		log.Println("RenameUser:", id, newname)
	}
	s.usersMutex.Lock()
	u, ok := s.users[id]
	if !ok {
		s.usersMutex.Unlock()
		return ErrUserNotFound
	}
	delete(s.userIDs, u.Name)
	u.Name = newname
	s.userIDs[newname] = id
	s.users[id] = u
	s.usersMutex.Unlock()
	return nil
}

// DeleteUser deletes a user object from the memoryStore
func (s *memoryStore) DeleteUser(id uint64) error {
	if storeDebug {
		log.Println("DeleteUser:", id)
	}
	s.usersMutex.Lock()
	u, ok := s.users[id]
	if !ok {
		s.usersMutex.Unlock()
		return ErrUserNotFound
	}
	delete(s.users, id)
	delete(s.userIDs, u.Name)
	s.usersMutex.Unlock()
	return nil
}

// ForEachUser ranges over all users from the memoryStore
func (s *memoryStore) ForEachUser(fn func(u *StoredUser) (del bool)) error {
	if storeDebug {
		log.Println("ForEachUser")
	}
	s.usersMutex.RLock()
	for k, v := range s.users {
		if fn(&v) {
			s.usersMutex.RUnlock()
			s.usersMutex.Lock()
			delete(s.users, k)
			s.usersMutex.Unlock()
			s.usersMutex.RLock()
		}
	}
	s.usersMutex.RUnlock()
	return nil
}

// type boltDBStore struct {
// 	db         *bolt.DB
// 	sessBucket []byte
// 	userBucket []byte
// }

// // NewBoltDBStore returns a Store that used the passed BoltDB as
// // a storage backend.
// func NewBoltDBStore(db *bolt.DB) (*Store, error) {
// 	err := db.Update(func(tx *bolt.Tx) error {
// 		_, err := tx.CreateBucketIfNotExists([]byte("users.S"))
// 		if err != nil {
// 			return err
// 		}
// 		_, err = tx.CreateBucketIfNotExists([]byte("users.U"))
// 		if err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	var s = boltDBStore{
// 		db:         db,
// 		sessBucket: []byte("users.S"),
// 		userBucket: []byte("users.U"),
// 	}
// 	return NewStore(&s), nil
// }

// // GetSession gets a Session object from the boltDBStore
// func (s *boltDBStore) GetSession(id string) (*StoredSession, error) {
// 	if storeDebug {
// 		log.Println("GetSession:", id)
// 	}
// 	var sess StoredSession
// 	err := s.db.View(func(tx *bolt.Tx) error {
// 		val := tx.Bucket(s.sessBucket).Get([]byte(id))
// 		if val == nil {
// 			return ErrSessionNotFound
// 		}
// 		return json.Unmarshal(val, &sess)
// 	})
// 	return &sess, err
// }

// // PutSession puts a Session object in the boltDBStore
// func (s *boltDBStore) PutSession(sess *StoredSession) error {
// 	if storeDebug {
// 		log.Println("PutSession:", sess.ID)
// 	}
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		val, err := json.Marshal(sess)
// 		if err != nil {
// 			return err
// 		}
// 		return tx.Bucket(s.sessBucket).Put([]byte(sess.ID), val)
// 	})
// }

// // DeleteSession deletes a session object from the boltDBStore
// func (s *boltDBStore) DeleteSession(id string) error {
// 	if storeDebug {
// 		log.Println("DeleteSession:", id)
// 	}
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		return tx.Bucket(s.sessBucket).Delete([]byte(id))
// 	})
// }

// // ForEachSession ranges over all sessions from the boltDBStore
// func (s *boltDBStore) ForEachSession(fn func(s *StoredSession) (del bool)) error {
// 	if storeDebug {
// 		log.Println("ForEachSession")
// 	}
// 	var sess StoredSession
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		return tx.Bucket(s.sessBucket).ForEach(func(k, v []byte) error {
// 			err := json.Unmarshal(v, &sess)
// 			if err != nil {
// 				return err
// 			}
// 			if fn(&sess) {
// 				err := tx.Bucket(s.sessBucket).Delete(k)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 			return nil
// 		})
// 	})
// }

// // GetUser gets a User object from the boltDBStore
// func (s *boltDBStore) GetUser(id uint64) (*StoredUser, error) {
// 	if storeDebug {
// 		log.Println("GetUser:", id)
// 	}
// 	var user StoredUser
// 	err := s.db.View(func(tx *bolt.Tx) error {
// 		val := tx.Bucket(s.userBucket).Get(itob(id))
// 		if val == nil {
// 			return ErrUserNotFound
// 		}
// 		return json.Unmarshal(val, &user)
// 	})
// 	return &user, err
// }

// // PutUser puts a User object in the boltDBStore
// func (s *boltDBStore) PutUser(u *StoredUser) error {
// 	if storeDebug {
// 		log.Println("PutUser:", u.Name)
// 	}
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		val, err := json.Marshal(u)
// 		if err != nil {
// 			return err
// 		}
// 		return tx.Bucket(s.userBucket).Put(itob(u.ID), val)
// 	})
// }

// // DeleteUser deletes a user object from the boltDBStore
// func (s *boltDBStore) DeleteUser(id uint64) error {
// 	if storeDebug {
// 		log.Println("DeleteUser:", id)
// 	}
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		return tx.Bucket(s.userBucket).Delete(itob(id))
// 	})
// }

// // ForEachUser ranges over all users from the boltDBStore
// func (s *boltDBStore) ForEachUser(fn func(u *StoredUser) (del bool)) error {
// 	if storeDebug {
// 		log.Println("ForEachUser")
// 	}
// 	var user StoredUser
// 	return s.db.Update(func(tx *bolt.Tx) error {
// 		return tx.Bucket(s.userBucket).ForEach(func(k, v []byte) error {
// 			err := json.Unmarshal(v, &user)
// 			if err != nil {
// 				return err
// 			}
// 			if fn(&user) {
// 				err := tx.Bucket(s.userBucket).Delete(k)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 			return nil
// 		})
// 	})
// }

// // itob returns an 8-byte big endian representation of v.
// func itob(v uint64) []byte {
// 	b := make([]byte, 8)
// 	binary.BigEndian.PutUint64(b, v)
// 	return b
// }

// // btoi returns an uint64 from a 8-byte slice.
// func btoi(v []byte) uint64 {
// 	if len(v) != 8 {
// 		log.Println("WARNING: btoi length is not 8 but", len(v))
// 	}
// 	return binary.BigEndian.Uint64(v)
// }
