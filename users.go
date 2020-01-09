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
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	defaultSessionCookieName               = "id"
	defaultSessionCookieExpirationLoggedin = time.Hour * 24 * 90
	defaultSessionCookieExpiration         = time.Minute
)

// ==================================================
// ===================== Errors =====================
// ==================================================

var (
	// ErrUserNotFound is returned when a store can't find the given user.
	ErrUserNotFound = errors.New("user not found")

	// ErrSessionNotFound is returned when a store can't find the given session.
	ErrSessionNotFound = errors.New("session not found")

	// ErrUserExists is returned when a new user with a username
	// that already exists is registered.
	ErrUserExists = errors.New("user already exists")

	// ErrLoginWrong is returned when login credentials are wrong.
	ErrLoginWrong = errors.New("login is wrong")

	// ErrNotLoggedIn is returned when a logged in user is expected.
	ErrNotLoggedIn = errors.New("not logged in")

	// ErrSessionGCRunning is returned when the session GC is already running.
	ErrSessionGCRunning = errors.New("session GC already running")

	// ErrSessionGCStopped is returned when the session GC is already stopped.
	ErrSessionGCStopped = errors.New("session GC already stopped")
)

// ==================================================
// ================= Main Interface =================
// ==================================================

type lookup int8

const (
	bySessionID lookup = iota + 1
	byUserID
	byUserName
	byCookie
)

type Accessor struct {
	lookup
	sessionID     string
	userID        uint64
	userName      string
	cookieRequest *http.Request
	cookieWriter  http.ResponseWriter
}

func BySessionID(id string) *Accessor {
	return &Accessor{
		lookup:    bySessionID,
		sessionID: id,
	}
}

func ByUserID(id uint64) *Accessor {
	return &Accessor{
		lookup: byUserID,
		userID: id,
	}
}

func ByUserName(name string) *Accessor {
	return &Accessor{
		lookup:   byUserName,
		userName: name,
	}
}

func ByCookie(w http.ResponseWriter, r *http.Request) *Accessor {
	return &Accessor{
		lookup:        byCookie,
		cookieRequest: r,
		cookieWriter:  w,
	}
}

// User is the type that is retuned from most Store methods. It contains
// the Name of the user, which is also the identification when it is stored.
// Salt is randomly generated on registration and used to salt the password
// hash which is then stored in Pass. The session that was used to retrieve
// this user is also embedded into the struct.
//
// The Data field can hold arbitrary application data which is saved using
// the Store.Save() method. To work with it use a type assertion.
type User struct {
	ID   uint64
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
	UserID     uint64
}

// Storage is implemented for different storage backends. The Get and Put
// methods need to be safe for use by multiple goroutines simultaneously.
// User IDs need to start at index 1, because 0 is reserved for errors.
type Storage interface {
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
	GetUser(id uint64) (*User, error)
	// If User is not found, error needs to be ErrUserNotFound
	GetUserID(username string) (uint64, error)
	// Put a User into the store
	PutUser(u *User) error
	// Add a User to the store and return the new user ID
	AddUser(u *User) (uint64, error)
	// Rename a User while keeping the ID the same
	RenameUser(id uint64, newname string) error
	// Delete a User from the store
	DeleteUser(id uint64) error
	// Run fn for each user and delete if true is returned
	ForEachUser(fn func(u *User) (del bool)) error
	// Return the number of saved users
	CountUsers() int
}

// Store is the main type of this library. It has a backend which can store
// users and sessions and provides all the relevant methods for working with
// them.
type Store struct {
	store     Storage
	stop      chan struct{}
	gcRunning bool
}

// NewStore creates a new store with a specified Storage backend. Only other
// libraries should call this function. Use New[...]Store() functions such as
// NewMemoryStore() instead. This function also starts a session GC that
// regularly deletes expired sessions.
func NewStore(s Storage) *Store {
	store := &Store{
		store:     s,
		stop:      make(chan struct{}, 1),
		gcRunning: true,
	}
	go store.sessionGC(store.stop)
	return store
}

func (s *Store) sessionGC(stop chan struct{}) {
	for {
		select {
		case <-time.After(defaultSessionCookieExpiration):
			count := 0
			s.store.ForEachSession(func(s *Session) (del bool) {
				if time.Now().After(s.Expires) {
					count++
					return true
				}
				return false
			})
			if count > 0 {
				log.Println("GCed", count, "sessions.")
			}
		case <-stop:
			s.gcRunning = false
			return
		}
	}
}

// StartSessionGC starts the session GC that regularly deletes expired
// sessions. It returns ErrSessionGCRunning if the GC is already running.
// When a new Store is created the sessionGC is automatically started.
func (s *Store) StartSessionGC() error {
	if s.gcRunning {
		return ErrSessionGCRunning
	}
	s.stop = make(chan struct{}, 1)
	s.gcRunning = true
	go s.sessionGC(s.stop)
	return nil
}

// StopSessionGC stops the session GC that regularly deletes expired
// sessions. It returns ErrSessionGCStopped if the GC is already stopped.
func (s *Store) StopSessionGC() error {
	if !s.gcRunning {
		return ErrSessionGCStopped
	}
	close(s.stop)
	return nil
}

// CountUsers returns the number of saved users
func (s *Store) CountUsers() int {
	return s.store.CountUsers()
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

// UserNameGet gets the User by its name. If the user
// does not exist ErrUserNotFound is returned.
func (s *Store) UserNameGet(username string) (*User, error) {
	id, err := s.store.GetUserID(username)
	if err != nil {
		return nil, err
	}
	return s.UserIDGet(id)
}

// UserIDGet gets the User by its ID. If the user
// does not exist ErrUserNotFound is returned.
func (s *Store) UserIDGet(id uint64) (*User, error) {
	user, err := s.store.GetUser(id)
	if err != nil {
		return nil, err
	}
	return user, nil
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
		user, err = s.store.GetUser(sess.UserID)
	} else {
		user = &User{}
	}
	if err != nil {
		changed2 := false
		if user == nil {
			user = &User{}
		}
		user.Session, changed2, err = s.logoutID(sess.ID)
		return user, changed || changed2, err
	}
	user.Session = sess
	return user, changed, nil
}

// CookieSaveData saves the passed data into the Data field of the User object
// linked to the current session. If no user is currently logged in
// ErrNotLoggedIn is returned.
func (s *Store) CookieSaveData(w http.ResponseWriter, r *http.Request, data interface{}) (*User, error) {
	user, changed, err := s.saveDataID(s.getCookieID(r), data)
	if changed {
		s.saveCookie(w, user.Session)
	}
	return user, err
}

// IDSaveData saves the passed data into the Data field of the User object
// linked to the specified session. If no user is currently logged in
// ErrNotLoggedIn is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDSaveData(id string, data interface{}) (*User, error) {
	user, _, err := s.saveDataID(id, data)
	return user, err
}

// UserNameSaveData saves the passed data into the Data field of the User object
// with the name specified in username. If the user
// does not exist ErrUserNotFound is returned.
func (s *Store) UserNameSaveData(username string, data interface{}) (*User, error) {
	id, err := s.store.GetUserID(username)
	if err != nil {
		return nil, err
	}
	return s.UserIDSaveData(id, data)
}

// UserIDSaveData saves the passed data into the Data field of the User object
// with the id specified in id. If the user
// does not exist ErrUserNotFound is returned.
func (s *Store) UserIDSaveData(id uint64, data interface{}) (*User, error) {
	u, err := s.userSaveData(id, data)
	return u, err
}

func (s *Store) userSaveData(id uint64, data interface{}) (*User, error) {
	u, err := s.store.GetUser(id)
	if err != nil {
		return nil, err
	}
	u.Data = data
	err = s.store.PutUser(u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) saveDataID(id string, data interface{}) (*User, bool, error) {
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
	if !sess.LoggedIn {
		return &User{Session: sess}, changed, ErrNotLoggedIn
	}
	user, err := s.userSaveData(sess.UserID, data)
	if err != nil {
		return &User{Session: sess}, changed, err
	}
	user.Session = sess
	return user, changed, nil
}

func (s *Store) getSessionID(id string) (*Session, bool, error) {
	sess, err := s.store.GetSession(id)
	if err != nil {
		if err == ErrSessionNotFound {
			sess, err := makeSession()
			return sess, true, err
		}
		return nil, false, err
	}
	if time.Now().After(sess.Expires) {
		sess, err = makeSession()
		return sess, true, err
	}
	sess.LastAccess = time.Now()
	if sess.LoggedIn {
		sess.Expires = time.Now().Add(defaultSessionCookieExpirationLoggedin)
	} else {
		sess.Expires = time.Now().Add(defaultSessionCookieExpiration)
	}
	return sess, true, nil
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
func (s *Store) CookieRegister(w http.ResponseWriter, r *http.Request, username, pass string) (*User, error) {
	u, changed, err := s.registerID(s.getCookieID(r), username, pass)
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
func (s *Store) IDRegister(id string, username, pass string) (*User, error) {
	u, _, err := s.registerID(id, username, pass)
	return u, err
}

// UserNameRegister registers a new user with a username and password. If the given
// username already exists ErrUserExists is returned.
func (s *Store) UserNameRegister(username, pass string) (*User, error) {
	_, err := s.store.GetUserID(username)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != ErrUserNotFound {
		return nil, err
	}

	var user = User{Name: username}

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
	uid, err := s.store.AddUser(&user)
	user.ID = uid
	user.UserID = uid
	if err != nil {
		return &user, err
	}
	return &user, nil
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
	_, err := s.store.GetUserID(name)
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
	uid, err := s.store.AddUser(&user)
	if err != nil {
		return &user, err
	}
	sess.LoggedIn = true
	sess.UserID = uid
	return &user, nil
}

// CookieSetUsername renames the current user to the new name. If the new
// username already exists ErrUserExists is returned. If there is no current
// user logged in ErrNotLoggedIn is returned.
func (s *Store) CookieSetUsername(w http.ResponseWriter, r *http.Request, nextusername string) (*User, error) {
	u, changed, err := s.setNameID(s.getCookieID(r), nextusername)
	if changed {
		s.saveCookie(w, u.Session)
	}
	return u, err
}

// IDSetUsername renames the current user to the new name. If the new
// username already exists ErrUserExists is returned. If there is no current
// user logged in ErrNotLoggedIn is returned.
//
// It is the callers responsibility to pass the session token (User.ID) back
// to the client.
func (s *Store) IDSetUsername(id string, nextusername string) (*User, error) {
	u, _, err := s.setNameID(id, nextusername)
	return u, err
}

// UserNameSetUsername renames the user to the new name. If the new
// username already exists ErrUserExists is returned.
func (s *Store) UserNameSetUsername(username, nextusername string) (*User, error) {
	id, err := s.store.GetUserID(username)
	if err != nil {
		return nil, err
	}
	return s.UserIDSetUsername(id, nextusername)
}

// UserIDSetUsername renames the user to the new name. If the new
// username already exists ErrUserExists is returned.
func (s *Store) UserIDSetUsername(id uint64, nextusername string) (*User, error) {
	user, err := s.store.GetUser(id)
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetUserID(nextusername)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != ErrUserNotFound {
		return nil, err
	}

	user.Name = nextusername

	err = s.store.DeleteUser(id)
	if err != nil {
		return user, err
	}

	err = s.store.PutUser(user)
	if err != nil {
		return user, err
	}
	return user, nil
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
	user, err := s.store.GetUser(sess.UserID)
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetUserID(name)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != ErrUserNotFound {
		return nil, err
	}

	err = s.store.RenameUser(sess.UserID, name)
	if err != nil {
		return user, err
	}
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

// UserNameSetPassword sets the password of the current user to a new one.
func (s *Store) UserNameSetPassword(username, pass string) (*User, error) {
	id, err := s.store.GetUserID(username)
	if err != nil {
		return nil, err
	}
	return s.UserIDSetPassword(id, pass)
}

// UserIDSetPassword sets the password of the user to a new one.
func (s *Store) UserIDSetPassword(id uint64, pass string) (*User, error) {
	user, err := s.store.GetUser(id)
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
	user, err := s.store.GetUser(sess.UserID)
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
func (s *Store) CookieLogin(w http.ResponseWriter, r *http.Request, username, pass string) (*User, error) {
	u, changed, err := s.loginID(s.getCookieID(r), username, pass)
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
func (s *Store) IDLogin(id string, username, pass string) (*User, error) {
	u, _, err := s.loginID(id, username, pass)
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
	uid, err := s.store.GetUserID(username)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, ErrLoginWrong
		}
		return nil, err
	}
	user, err := s.store.GetUser(uid)
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
		sess.UserID = user.ID
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

// UserIDDelete deletes the user with the given user ID. It
// returns ErrUserNotFound if there is no such user stored.
func (s *Store) UserIDDelete(id uint64) (*User, error) {
	err := s.store.DeleteUser(id)
	return nil, err
}

// UserNameDelete deletes the user with the given username. It
// returns ErrUserNotFound if there is no such user stored.
func (s *Store) UserNameDelete(username string) (*User, error) {
	id, err := s.store.GetUserID(username)
	if err != nil {
		return nil, err
	}
	return s.UserIDDelete(id)
}

func (s *Store) deleteID(id string) (*Session, bool, error) {
	sess, changed, err := s.getSessionID(id)
	if err != nil {
		return sess, changed, err
	}
	if sess.LoggedIn == false {
		err = ErrNotLoggedIn
	} else {
		err = s.store.DeleteUser(sess.UserID)
		if err == nil {
			sess.LoggedIn = false
			sess.UserID = 0
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
