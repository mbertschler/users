package users

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	SessionCookieName       = "id"
	SessionCookieExpiration = time.Hour * 24 * 90
	SessionDebug            = true
)

// ==================================================
// ===================== Errors =====================
// ==================================================

var (
	UserExists      = errors.New("User already exists")
	UserNotFound    = errors.New("User not found")
	SessionNotFound = errors.New("Session not found")
	ServerError     = errors.New("User server error")
	LoginWrong      = errors.New("Login is wrong")
	NotLoggedIn     = errors.New("Not logged in")
)

// ==================================================
// ================= Main Interface =================
// ==================================================

type Store struct {
	store Storer
}

type Storer interface {
	GetSession(id string) (*Session, error)
	PutSession(s *Session) error

	GetUser(name string) (*User, error)
	PutUser(u *User) error
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
	user, err := s.getUser(sess)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	user.Session = sess
	return user, changed, nil
}

func (s *Store) getUser(sess *Session) (*User, error) {
	if sess.LoggedIn && sess.Bound {
		u, err := s.store.GetUser(sess.User)
		if err == nil {
			return u, nil
		} else {
			return nil, err
		}
	}
	return nil, NotLoggedIn
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
		if err == SessionNotFound {
			if SessionDebug {
				log.Println("Not found:", id[:10], "making new")
			}
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
			if SessionDebug {
				log.Println("Creating new session")
			}
			sess, err := makeSession()
			return sess, true, err
		}
		return nil, false, err
	}
	if SessionDebug {
		//log.Println("Loading session from MemoryStore")
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
		return nil, UserExists
	} else {
		if err != UserNotFound {
			return nil, err
		}
	}

	var user = User{Name: name}

	user.Salt = make([]byte, 32)
	_, err = rand.Read(user.Salt)
	if err != nil {
		log.Println("Rand error:", err)
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
	sess.Bound = true
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
		if err == UserNotFound {
			return nil, LoginWrong
		} else {
			return nil, err
		}
	}
	dk, err := scrypt.Key([]byte(password), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(dk, user.Pass) {
		sess.LoggedIn = true
		sess.Bound = true
		sess.User = username
		return user, nil
	}
	sess.LoggedIn = false
	return nil, LoginWrong
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
		return sess, changed, NotLoggedIn
	}
	err = s.logout(sess)
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

func (s *Store) logout(sess *Session) error {
	if sess.LoggedIn == false || sess.Bound == false {
		return NotLoggedIn
	}
	sess.LoggedIn = false
	return nil
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
	Bound    bool
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
	if SessionDebug {
		log.Println("Sess created: ", str[:10])
	}
	expiration := time.Now().Add(SessionCookieExpiration)
	s := Session{
		ID:      str,
		Expires: expiration,
		LastCon: time.Now(),
	}
	return &s, nil
}

func DecodeUser(v []byte) (*User, error) {
	user := new(User)
	err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u User) Encode() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(&u)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
