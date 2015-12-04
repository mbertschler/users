package users

import (
	"bytes"
	"crypto/rand"
	"log"
	"net/http"

	"golang.org/x/crypto/scrypt"
)

// const (
// 	DefaultSessionBucket = "users.S"
// 	DefaultUserBucket    = "users.U"
// )

type MemoryStore struct {
	sessions map[string]Session
	users    map[string]User
	path     string
}

func NewMemoryStore(path string) *MemoryStore {
	var s = MemoryStore{
		sessions: make(map[string]Session),
		users:    make(map[string]User),
		path:     path,
	}
	return &s
}

func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) GetSession(r *http.Request) (*Session, bool, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			if SessionDebug {
				log.Println("Creating new session")
			}
			sess, err := makeSession()
			return sess, false, err
		}
		return nil, false, err
	}
	if SessionDebug {
		//log.Println("Loading session from MemoryStore")
	}
	sess, ok := s.sessions[cookie.Value]
	// TODO check if session is expired
	if !ok {
		if SessionDebug {
			log.Println("Not found:    ", cookie.Value[:10])
			log.Println("Didn't find session - creating new")
		}
		sess, err := makeSession()
		return sess, false, err
	}

	return &sess, true, nil
}

func (s *MemoryStore) SaveSession(w http.ResponseWriter, sess *Session) error {
	cookie := http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     s.path,
		HttpOnly: true,
		Expires:  sess.Expires,
	}
	http.SetCookie(w, &cookie)
	s.sessions[sess.ID] = *sess
	return nil
}

func (s *MemoryStore) Register(sess *Session, name, pass string) (*User, error) {
	_, ok := s.users[name]
	if ok {
		return nil, UserExists
	}

	var user = User{Name: name}

	user.Salt = make([]byte, 32)
	_, err := rand.Read(user.Salt)
	if err != nil {
		log.Println("Rand error:", err)
		return nil, err
	}

	user.Pass, err = scrypt.Key([]byte(pass), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	s.users[name] = user
	sess.LoggedIn = true
	sess.Bound = true
	sess.User = name
	return &user, nil
}

func (s *MemoryStore) Login(sess *Session, username, password string) (*User, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, LoginWrong
	}
	dk, err := scrypt.Key([]byte(password), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(dk, user.Pass) {
		sess.LoggedIn = true
		sess.Bound = true
		sess.User = username
		return &user, nil
	}
	sess.LoggedIn = false
	return nil, LoginWrong
}

func (s *MemoryStore) Logout(sess *Session) error {
	sess.LoggedIn = false
	return nil
}

func (s *MemoryStore) GetUser(sess *Session) (*User, error) {
	if sess.LoggedIn && sess.Bound {
		u, ok := s.users[sess.User]
		if ok {
			return &u, nil
		}
		return nil, UserNotFound
	}
	return nil, NotLoggedIn
}

func (s *MemoryStore) SaveUser(u *User) error {
	s.users[u.Name] = *u
	return nil
}

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
