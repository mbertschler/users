package users

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	SessionCookieName       = "id"
	SessionCookieExpiration = time.Hour * 24 * 90
)

// ==================================================
// ===================== Errors =====================
// ==================================================

var (
	// ErrUserNotFound is returned when a store can't find the given user
	ErrUserNotFound = errors.New("User not found")

	// ErrSessionNotFound is returned when a store can't find the given session
	ErrSessionNotFound = errors.New("Session not found")

	// ErrUserExists is returned when a new user with a username
	// that already exists is registered
	ErrUserExists = errors.New("User already exists")

	// ErrLoginWrong is returned when login credentials are wrong
	ErrLoginWrong = errors.New("Login is wrong")

	// ErrNotLoggedIn is returned when a logged in user is expected
	ErrNotLoggedIn = errors.New("Not logged in")
)

// ==================================================
// ================= Main Interface =================
// ==================================================

// Storer is implemented for different storage backends. The Get and Put
// methods need to be safe for use by multiple goroutines simultaneously.
type Storer interface {
	// Get a Session from the store
	// If Session is not found, error needs to be ErrSessionNotFound
	GetSession(id string) (*Session, error)
	// Put a Session into the store
	PutSession(s *Session) error

	// Get a User from the store
	// If User is not found, error needs to be ErrUserNotFound
	GetUser(name string) (*User, error)
	// Put a User into the store
	PutUser(u *User) error
}

// Store has all the methods that are called from users of this library.
type Store struct {
	store Storer
}

func (s *Store) Get(w http.ResponseWriter, r *http.Request) (*User, error) {
	user, changed, err := s.getID(s.getCookieID(r))
	if changed {
		s.saveCookie(w, user.Session)
	}
	return user, err
}

func (s *Store) GetID(id string) (*User, error) {
	user, _, err := s.getID(id)
	return user, err
}

func (s *Store) getID(id string) (*User, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	if changed {
		err = s.store.PutSession(sess)
		if err != nil {
			return &User{Session: sess}, changed, err
		}
	}
	var user *User
	if sess.LoggedIn {
		user, err = s.store.GetUser(sess.User)
	}
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	user.Session = sess
	return user, changed, nil
}

func (s *Store) Save(u *User) error {
	user := *u
	user.Session = nil
	return s.store.PutUser(u)
}

func (s *Store) getSessionID(id string) (*Session, bool, error) {
	sess, err := s.store.GetSession(id)
	// TODO check if session is expired
	if err != nil {
		if err == ErrSessionNotFound {
			sess, err := makeSession()
			return sess, true, err
		}
		return nil, false, err
	}
	return sess, false, nil
}

func (s *Store) getSession(r *http.Request) (*Session, bool, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			sess, err := makeSession()
			return sess, true, err
		}
		return nil, false, err
	}
	return s.getSessionID(cookie.Value)
}

func (s *Store) getCookieID(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Store) saveSession(w http.ResponseWriter, sess *Session) error {
	cookie := http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Expires:  sess.Expires,
	}
	http.SetCookie(w, &cookie)
	return s.store.PutSession(sess)
}

func (s *Store) saveCookie(w http.ResponseWriter, sess *Session) {
	cookie := http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Expires:  sess.Expires,
	}
	http.SetCookie(w, &cookie)
}

func (s *Store) Register(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error) {
	u, changed, err := s.registerID(s.getCookieID(r), user, pass)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

func (s *Store) RegisterID(id string, user, pass string) (*User, error) {
	u, _, err := s.registerID(id, user, pass)
	return u, err
}

func (s *Store) registerID(id string, user, pass string) (*User, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u, err := s.register(sess, user, pass)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u.Session = sess
	err = s.store.PutSession(sess)
	changed = true
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	return u, changed, nil
}

func (s *Store) register(sess *Session, name, pass string) (*User, error) {
	_, err := s.store.GetUser(name)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != ErrUserNotFound {
		return nil, err
	}

	var user = User{Name: name}

	user.Salt = make([]byte, 32)
	_, err = rand.Read(user.Salt)
	if err != nil {
		return nil, err
	}

	user.Pass, err = scrypt.Key([]byte(pass), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	err = s.store.PutUser(&user)
	if err != nil {
		return &user, err
	}
	sess.LoggedIn = true
	sess.User = name
	return &user, nil
}

func (s *Store) Login(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error) {
	u, changed, err := s.loginID(s.getCookieID(r), user, pass)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

func (s *Store) LoginID(id string, user, pass string) (*User, error) {
	u, _, err := s.loginID(id, user, pass)
	return u, err
}

func (s *Store) loginID(id string, user, pass string) (*User, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u, err := s.login(sess, user, pass)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u.Session = sess
	err = s.store.PutSession(sess)
	changed = true
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	return u, changed, nil
}

func (s *Store) login(sess *Session, username, password string) (*User, error) {
	user, err := s.store.GetUser(username)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrLoginWrong
		}
		return nil, err
	}
	dk, err := scrypt.Key([]byte(password), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(dk, user.Pass) {
		sess.LoggedIn = true
		sess.User = username
		return user, nil
	}
	sess.LoggedIn = false
	return nil, ErrLoginWrong
}

func (s *Store) Logout(w http.ResponseWriter, r *http.Request) error {
	sess, changed, err := s.logoutID(s.getCookieID(r))
	if changed {
		s.saveCookie(w, sess)
	}
	return err
}

func (s *Store) LogoutID(id string) error {
	_, _, err := s.logoutID(id)
	return err
}

func (s *Store) logoutID(id string) (*Session, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return sess, changed, err
	}
	if changed {
		return sess, changed, ErrNotLoggedIn
	}
	if sess.LoggedIn == false {
		err = ErrNotLoggedIn
	} else {
		sess.LoggedIn = false
	}
	if err != nil {
		return sess, changed, err
	}
	err = s.store.PutSession(sess)
	changed = true
	if err != nil {
		return sess, changed, err
	}
	return sess, changed, nil
}

// ==================================================
// ====================== Types =====================
// ==================================================

type User struct {
	Name string
	Pass []byte
	Salt []byte
	Data interface{}
	*Session
}

type Session struct {
	ID       string
	LoggedIn bool
	Expires  time.Time
	LastCon  time.Time
	User     string // key for user bucket
}

func makeSession() (*Session, error) {
	buf := make([]byte, 24)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	str := base64.StdEncoding.EncodeToString(buf)
	expiration := time.Now().Add(SessionCookieExpiration)
	s := Session{
		ID:      str,
		Expires: expiration,
		LastCon: time.Now(),
	}
	return &s, nil
}

// func DecodeUser(v []byte) (*User, error) {
// 	user := new(User)
// 	err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(user)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return user, nil
// }

// func (u User) Encode() ([]byte, error) {
// 	var buf bytes.Buffer
// 	err := gob.NewEncoder(&buf).Encode(&u)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return buf.Bytes(), nil
// }
