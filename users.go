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
	defaultSessionCookieName       = "id"
	defaultSessionCookieExpiration = time.Hour * 24 * 90
)

// ==================================================
// ===================== Errors =====================
// ==================================================

var (
	// ErrUserNotFound is returned when a store can't find the given user.
	ErrUserNotFound = errors.New("User not found")

	// ErrSessionNotFound is returned when a store can't find the given session.
	ErrSessionNotFound = errors.New("Session not found")

	// ErrUserExists is returned when a new user with a username
	// that already exists is registered.
	ErrUserExists = errors.New("User already exists")

	// ErrLoginWrong is returned when login credentials are wrong.
	ErrLoginWrong = errors.New("Login is wrong")

	// ErrNotLoggedIn is returned when a logged in user is expected.
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
	// Delete a Session from the store
	DeleteSession(id string) error
	// Run fn for each session and delete if true is returned
	ForEachSession(fn func(s *Session) (del bool)) error

	// Get a User from the store
	// If User is not found, error needs to be ErrUserNotFound
	GetUser(name string) (*User, error)
	// Put a User into the store
	PutUser(u *User) error
	// Delete a User from the store
	DeleteUser(username string) error
	// Run fn for each user and delete if true is returned
	ForEachUser(fn func(u *User) (del bool)) error
}

// Store is the main type of this library. It has a backend which can store
// users and sessions and provides all the relevant methods for working with
// them.
type Store struct {
	store Storer
}

// NewStore creates a new store with a specified Storer backend. Only other
// libraries should call this function. Use New[...]Store() functions such as
// NewMemoryStore() instead.
func NewStore(s Storer) *Store {
	return &Store{s}
}

// CookieGet gets the User associated with the current client.
// If there is no session cookie set in the request or the session is expired
// or not valid anymore, a new session cookie is created and set.
// If no user is logged in with this session the nil value of User with the
// embedded Session is returned.
func (s *Store) CookieGet(w http.ResponseWriter, r *http.Request) (*User, error) {
	user, changed, err := s.getID(s.getCookieID(r))
	if changed {
		s.saveCookie(w, user.Session)
	}
	return user, err
}

// IDGet gets the User associated with a session ID.
// If there is no session with this ID or the session expired,
// a new session is created.
// If no user is logged in with this session the nil value of User with the
// embedded Session is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDGet(id string) (*User, error) {
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
	} else {
		user = &User{}
	}
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	user.Session = sess
	return user, changed, nil
}

// UserSaveData saves the passed data into the Data field of the User object
// with the name specified in username. If the user
// does not exist ErrUserNotFound is returned.
func (s *Store) UserSaveData(username string, data interface{}) (*User, error) {
	u, err := s.store.GetUser(username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	u.Data = data
	err = s.store.PutUser(u)
	if err != nil {
		return nil, err
	}
	return u, nil
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
	cookie, err := r.Cookie(defaultSessionCookieName)
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
	cookie, err := r.Cookie(defaultSessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Store) saveSession(w http.ResponseWriter, sess *Session) error {
	cookie := http.Cookie{
		Name:     defaultSessionCookieName,
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
		Name:     defaultSessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Expires:  sess.Expires,
	}
	http.SetCookie(w, &cookie)
}

// CookieRegister registers a new user with a username and password. If the given
// username already exists ErrUserExists is returned.
func (s *Store) CookieRegister(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error) {
	u, changed, err := s.registerID(s.getCookieID(r), user, pass)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

// IDRegister registers a new user with a username and password. If the given
// username already exists ErrUserExists is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDRegister(id string, user, pass string) (*User, error) {
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

	//start := time.Now()
	user.Pass, err = scrypt.Key([]byte(pass), user.Salt, 16384, 8, 1, 32)
	//log.Println("scrypt.Key Register took:", time.Now().Sub(start))
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

// CookieSetName renames the current user to the new name. If the new
// username already exists ErrUserExists is returned. If there is no current
// user logged in ErrNotLoggedIn is returned.
func (s *Store) CookieSetName(w http.ResponseWriter, r *http.Request, name string) (*User, error) {
	u, changed, err := s.setNameID(s.getCookieID(r), name)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

// IDSetName renames the current user to the new name. If the new
// username already exists ErrUserExists is returned. If there is no current
// user logged in ErrNotLoggedIn is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDSetName(id string, name string) (*User, error) {
	u, _, err := s.setNameID(id, name)
	return u, err
}

func (s *Store) setNameID(id string, name string) (*User, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u, err := s.setName(sess, name)
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

func (s *Store) setName(sess *Session, name string) (*User, error) {
	if !sess.LoggedIn {
		return nil, ErrNotLoggedIn
	}
	user, err := s.store.GetUser(sess.User)
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetUser(name)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != ErrUserNotFound {
		return nil, err
	}

	err = s.store.DeleteUser(user.Name)
	if err != nil {
		return user, err
	}

	user.Name = name
	err = s.store.PutUser(user)
	if err != nil {
		return user, err
	}

	sess.User = name
	return user, nil
}

// CookieSetPassword sets the password of the current user to a new one. If
// there is no current user logged in ErrNotLoggedIn is returned.
func (s *Store) CookieSetPassword(w http.ResponseWriter, r *http.Request, pass string) (*User, error) {
	u, changed, err := s.setPasswordID(s.getCookieID(r), pass)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

// IDSetPassword sets the password of the current user to a new one. If
// there is no current user logged in ErrNotLoggedIn is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDSetPassword(id string, pass string) (*User, error) {
	u, _, err := s.setPasswordID(id, pass)
	return u, err
}

func (s *Store) setPasswordID(id string, pass string) (*User, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	u, err := s.setPassword(sess, pass)
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

func (s *Store) setPassword(sess *Session, pass string) (*User, error) {
	if !sess.LoggedIn {
		return nil, ErrNotLoggedIn
	}
	user, err := s.store.GetUser(sess.User)
	if err != nil {
		return nil, err
	}

	user.Salt = make([]byte, 32)
	_, err = rand.Read(user.Salt)
	if err != nil {
		return nil, err
	}

	user.Pass, err = scrypt.Key([]byte(pass), user.Salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	err = s.store.PutUser(user)
	if err != nil {
		return user, err
	}
	return user, nil
}

// CookieLogin logs a user in with a username and password. If the credentials for
// the login are wrong, ErrLoginWrong is returned.
func (s *Store) CookieLogin(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error) {
	u, changed, err := s.loginID(s.getCookieID(r), user, pass)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

// IDLogin logs a user in with a username and password. If the credentials for
// the login are wrong, ErrLoginWrong is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDLogin(id string, user, pass string) (*User, error) {
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
	// start := time.Now()
	dk, err := scrypt.Key([]byte(password), user.Salt, 16384, 8, 1, 32)
	// log.Println("scrypt.Key Login took:", time.Now().Sub(start))
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

// CookieLogout logs the user that is associated with this client. It
// returns ErrNotLoggedIn if no user is currently logged in.
func (s *Store) CookieLogout(w http.ResponseWriter, r *http.Request) (*User, error) {
	sess, changed, err := s.logoutID(s.getCookieID(r))
	if changed {
		s.saveCookie(w, sess)
	}
	return &User{Session: sess}, err
}

// IDLogout logs the user that is associated with this session id out. It
// returns ErrNotLoggedIn if no user is currently logged in.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDLogout(id string) (*User, error) {
	sess, _, err := s.logoutID(id)
	return &User{Session: sess}, err
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

// CookieDelete deletes the user that is associated with this client. It
// returns ErrNotLoggedIn if no user is currently logged in.
func (s *Store) CookieDelete(w http.ResponseWriter, r *http.Request) (*User, error) {
	sess, changed, err := s.deleteID(s.getCookieID(r))
	if changed {
		s.saveCookie(w, sess)
	}
	return &User{Session: sess}, err
}

// IDDelete deltes the user that is associated with this session id. It
// returns ErrNotLoggedIn if no user is currently logged in.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDDelete(id string) (*User, error) {
	sess, _, err := s.deleteID(id)
	return &User{Session: sess}, err
}

func (s *Store) deleteID(id string) (*Session, bool, error) {
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
		err = s.store.DeleteUser(sess.User)
		if err == nil {
			sess.LoggedIn = false
			sess.User = ""
		}
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

// User is the type that is retuned from most Store methods. It contains the
// Name of the user, which is also the identification when it is stored.
// Salt is randomly generated on registration and used to salt the password
// hash which is then stored in Pass. The Session that was used to retrieve
// this user is also embedded into the struct.
//
// The Data field can hold arbitrary application data which is saved using
// the Store.Save() method. To work with it use a type assertion.
type User struct {
	Name string
	Pass []byte
	Salt []byte
	Data interface{}
	*Session
}

// Session is embedded into the User object. It is identified by its random
// ID token which is base64 encoded. It also tracks expiration time and last
// access time. If a user is logged in with this session, LoggedIn is true
// and User holds a username. After a logout User still holds the username.
type Session struct {
	ID         string
	Expires    time.Time
	LastAccess time.Time
	LoggedIn   bool
	User       string
}

// make a new session with 24 random bytes which results in 32 base64 bytes
func makeSession() (*Session, error) {
	buf := make([]byte, 24)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	str := base64.StdEncoding.EncodeToString(buf)
	expiration := time.Now().Add(defaultSessionCookieExpiration)
	s := Session{
		ID:         str,
		Expires:    expiration,
		LastAccess: time.Now(),
	}
	return &s, nil
}
