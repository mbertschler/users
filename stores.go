package users

// const (
// 	DefaultSessionBucket = "users.S"
// 	DefaultUserBucket    = "users.U"
// )

type MemoryStore struct {
	sessions map[string]Session
	users    map[string]User
}

func NewMemoryStore() *Store {
	var s = MemoryStore{
		sessions: make(map[string]Session),
		users:    make(map[string]User),
	}
	return &Store{&s}
}

func (s *MemoryStore) GetSession(id string) (*Session, error) {
	sess, ok := s.sessions[id]
	if !ok {
		return nil, SessionNotFound
	}
	return &sess, nil
}

func (s *MemoryStore) PutSession(sess *Session) error {
	s.sessions[sess.ID] = *sess
	return nil
}

func (s *MemoryStore) GetUser(name string) (*User, error) {
	u, ok := s.users[name]
	if !ok {
		return nil, UserNotFound
	}
	return &u, nil
}

func (s *MemoryStore) PutUser(u *User) error {
	s.users[u.Name] = *u
	return nil
}
