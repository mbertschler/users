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
	UserExists   = errors.New("User already exists")
	UserNotFound = errors.New("User not found")
	ServerError  = errors.New("User server error")
	LoginWrong   = errors.New("Login is wrong")
	NotLoggedIn  = errors.New("Not logged in")
)

// ==================================================
// ================= Main Interface =================
// ==================================================

type Store interface {
	Get(w http.ResponseWriter, r *http.Request) (*User, error)
	Register(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error)
	Login(w http.ResponseWriter, r *http.Request, user, pass string) (*User, error)
	Logout(w http.ResponseWriter, r *http.Request) error

	GetID(id string) (*User, error)
	RegisterID(id string, user, pass string) (*User, error)
	LoginID(id string, user, pass string) (*User, error)
	LogoutID(id string) error

	Save(u *User) error
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
	buf := make([]byte, 66)
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
