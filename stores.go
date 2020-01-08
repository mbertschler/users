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
