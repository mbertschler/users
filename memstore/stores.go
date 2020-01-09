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

package memstore

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/mbertschler/crowd"
)

// enable debug messages when store functions are called
const storeDebug = false

// MemStore is a thread safe memory backend for the Store type. It
// implements the Storer interface and provides user and session storage.
// Do not use this directly, instead call NewMemStore().
// memoryStore saves the actual values behind the passed pointers.
type MemStore struct {
	sessions      map[string]crowd.Session
	sessionsMutex sync.RWMutex
	users         map[uint64]crowd.User
	usersMutex    sync.RWMutex
	userIDs       map[string]uint64
	maxUserID     uint64
}

// NewMemStore returns a Store with a memory backend.
func NewMemStore() *MemStore {
	return &MemStore{
		sessions: make(map[string]crowd.Session),
		users:    make(map[uint64]crowd.User),
		userIDs:  make(map[string]uint64),
	}
}

func (s *MemStore) nextUserID() uint64 {
	return atomic.AddUint64(&s.maxUserID, 1)
}

// CountUsers returns the number of saved users
func (s *MemStore) CountUsers() int {
	if storeDebug {
		log.Println("CountUsers")
	}
	s.usersMutex.RLock()
	count := len(s.users)
	s.usersMutex.RUnlock()
	return count
}

// GetSession gets a Session object from the memoryStore
func (s *MemStore) GetSession(id string) (*crowd.Session, error) {
	if storeDebug {
		log.Println("GetSession:", id)
	}
	s.sessionsMutex.RLock()
	sess, ok := s.sessions[id]
	s.sessionsMutex.RUnlock()
	if !ok {
		return nil, crowd.ErrSessionNotFound
	}
	return &sess, nil
}

// PutSession puts a Session object in the memoryStore
func (s *MemStore) PutSession(sess *crowd.Session) error {
	if storeDebug {
		log.Println("PutSession:", sess.ID)
	}
	s.sessionsMutex.Lock()
	s.sessions[sess.ID] = *sess
	s.sessionsMutex.Unlock()
	return nil
}

// DeleteSession deletes a session object from the memoryStore
func (s *MemStore) DeleteSession(id string) error {
	if storeDebug {
		log.Println("DeleteSession:", id)
	}
	s.sessionsMutex.Lock()
	delete(s.sessions, id)
	s.sessionsMutex.Unlock()
	return nil
}

// ForEachSession ranges over all sessions from the memoryStore
func (s *MemStore) ForEachSession(fn func(s *crowd.Session) (del bool)) error {
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
func (s *MemStore) GetUser(id uint64) (*crowd.User, error) {
	if storeDebug {
		log.Println("GetUser:", id)
	}
	s.usersMutex.RLock()
	u, ok := s.users[id]
	s.usersMutex.RUnlock()
	if !ok {
		return nil, crowd.ErrUserNotFound
	}
	return &u, nil
}

// GetUserID gets the user ID via the username from the memoryStore
func (s *MemStore) GetUserID(username string) (uint64, error) {
	if storeDebug {
		log.Println("GetUserID:", username)
	}
	s.usersMutex.RLock()
	uid, ok := s.userIDs[username]
	s.usersMutex.RUnlock()
	if !ok {
		return 0, crowd.ErrUserNotFound
	}
	return uid, nil
}

// PutUser puts a User object in the memoryStore
func (s *MemStore) PutUser(u *crowd.User) error {
	if storeDebug {
		log.Println("PutUser:", u.ID, u.Name)
	}
	s.usersMutex.Lock()
	s.users[u.ID] = *u
	s.usersMutex.Unlock()
	return nil
}

// AddUser puts a new User object in the memoryStore and returns the user ID
func (s *MemStore) AddUser(u *crowd.User) (uint64, error) {
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
func (s *MemStore) RenameUser(id uint64, newname string) error {
	if storeDebug {
		log.Println("RenameUser:", id, newname)
	}
	s.usersMutex.Lock()
	u, ok := s.users[id]
	if !ok {
		s.usersMutex.Unlock()
		return crowd.ErrUserNotFound
	}
	delete(s.userIDs, u.Name)
	u.Name = newname
	s.userIDs[newname] = id
	s.users[id] = u
	s.usersMutex.Unlock()
	return nil
}

// DeleteUser deletes a user object from the memoryStore
func (s *MemStore) DeleteUser(id uint64) error {
	if storeDebug {
		log.Println("DeleteUser:", id)
	}
	s.usersMutex.Lock()
	u, ok := s.users[id]
	if !ok {
		s.usersMutex.Unlock()
		return crowd.ErrUserNotFound
	}
	delete(s.users, id)
	delete(s.userIDs, u.Name)
	s.usersMutex.Unlock()
	return nil
}

// ForEachUser ranges over all users from the memoryStore
func (s *MemStore) ForEachUser(fn func(u *crowd.User) (del bool)) error {
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
