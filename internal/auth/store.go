package auth

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type User struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	KeyHash string `yaml:"key_hash"`
	Enabled bool   `yaml:"enabled"`
}

type usersFile struct {
	Users []User `yaml:"users"`
}

type UserStore struct {
	mu      sync.RWMutex
	users   map[string]*User
	hashIdx map[[32]byte]string
}

func NewUserStore() *UserStore {
	return &UserStore{
		users:   make(map[string]*User),
		hashIdx: make(map[[32]byte]string),
	}
}

func (s *UserStore) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading users file: %w", err)
	}

	var f usersFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing users file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = make(map[string]*User, len(f.Users))
	s.hashIdx = make(map[[32]byte]string, len(f.Users))

	for i := range f.Users {
		u := &f.Users[i]
		s.users[u.ID] = u

		hashBytes, err := hex.DecodeString(u.KeyHash)
		if err != nil || len(hashBytes) != 32 {
			return fmt.Errorf("invalid key_hash for user %q: must be 64 hex chars", u.ID)
		}
		var hash [32]byte
		copy(hash[:], hashBytes)
		s.hashIdx[hash] = u.ID
	}

	return nil
}

func (s *UserStore) Authenticate(rawKey string) (*User, bool) {
	computed := HashAPIKey(rawKey)

	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.hashIdx[computed]
	if !ok {
		return nil, false
	}

	user := s.users[userID]
	if user == nil || !user.Enabled {
		return nil, false
	}

	return user, true
}

func (s *UserStore) GetUser(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *UserStore) UserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}
