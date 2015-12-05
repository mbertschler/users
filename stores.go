package users

import (
	"log"
	"sync"
)

const storeDebug = false

// MemoryStore is a thread safe memory backend for the Store type. It
// implements the Storer interface and provides user and session storage.
type MemoryStore struct {
	sessions      map[string]Session
	sessionsMutex sync.RWMutex
	users         map[string]User
	usersMutex    sync.RWMutex
}

// NewMemoryStore returns a Store with an initialized MemoryStore backend
func NewMemoryStore() *Store {
	var s = MemoryStore{
		sessions: make(map[string]Session),
		users:    make(map[string]User),
	}
	return &Store{&s}
}

// GetSession gets a Session object from the MemoryStore
func (s *MemoryStore) GetSession(id string) (*Session, error) {
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

// PutSession puts a Session object in the MemoryStore
func (s *MemoryStore) PutSession(sess *Session) error {
	if storeDebug {
		log.Println("PutSession:", sess.ID)
	}
	s.sessionsMutex.Lock()
	s.sessions[sess.ID] = *sess
	s.sessionsMutex.Unlock()
	return nil
}

// GetUser gets a User object from the MemoryStore
func (s *MemoryStore) GetUser(name string) (*User, error) {
	if storeDebug {
		log.Println("GetUser:", name)
	}
	s.usersMutex.RLock()
	u, ok := s.users[name]
	s.usersMutex.RUnlock()
	if !ok {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// PutUser puts a User object in the MemoryStore
func (s *MemoryStore) PutUser(u *User) error {
	if storeDebug {
		log.Println("PutUser:", u.Name)
	}
	s.usersMutex.Lock()
	s.users[u.Name] = *u
	s.usersMutex.Unlock()
	return nil
}

// const (
// 	DefaultSessionBucket = "users.S"
// 	DefaultUserBucket    = "users.U"
// )

// type BoltDBStore struct {
// 	db *bolt.DB
// }

// func NewBoltDBStore(path string, mode os.FileMode, options *bolt.Options) *BoltDBStore {
// 	var s = BoltDBStore{
// 		users:    make(map[string]User),
// 		sessions: make(map[string]Session),
// 	}
// 	return &s
// }

// type OpenBoltDBStore struct {
// 	db            *bolt.DB
// 	sessionBucket string
// 	userBucket    string
// }

// func NewOpenBoltDBStore(db *bolt.DB, sessionBucket, userBucket string) *OpenBoltDBStore {
// 	var s = OpenBoltDBStore{
// 		users:    make(map[string]User),
// 		sessions: make(map[string]Session),
// 	}
// 	return &s
// }

// func loadDB() {
// 	var err error
// 	db, err := bolt.Open(config.DBPath, 0644, &bolt.Options{Timeout: 1 * time.Second})
// 	if err != nil {
// 		log.Fatal("Could not open DB at", config.DBPath, ": ", err)
// 	}
// 	db.Update(func(tx *bolt.Tx) error {
// 		_, err = tx.CreateBucketIfNotExists(BucketApp)
// 		if err != nil {
// 			log.Fatalln("Bucket app could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketEvents)
// 		if err != nil {
// 			log.Fatalln("Bucket events could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketProjects)
// 		if err != nil {
// 			log.Fatalln("Bucket projects could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketJury)
// 		if err != nil {
// 			log.Fatalln("Bucket jury could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketSMS)
// 		if err != nil {
// 			log.Fatalln("Bucket sms could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketSessions)
// 		if err != nil {
// 			log.Fatalln("Bucket sms could not be created")
// 		}
// 		return nil
// 	})
// }

// func B(in string) []byte {
// 	return []byte(in)
// }

// func IDtoBytes(id uint64) []byte {
// 	idbytes := make([]byte, 8)
// 	binary.BigEndian.PutUint64(idbytes, id)
// 	return idbytes
// }

// func BytesToID(b []byte) uint64 {
// 	return binary.BigEndian.Uint64(b)
// }

// func TimeToBytes(t time.Time) []byte {
// 	buf := make([]byte, 8)
// 	binary.BigEndian.PutUint64(buf, uint64(t.UnixNano()))
// 	return buf
// }
// func BytesToTime(b []byte) time.Time {
// 	unixnano := binary.BigEndian.Uint64(b)
// 	return time.Unix(0, int64(unixnano))
// }
