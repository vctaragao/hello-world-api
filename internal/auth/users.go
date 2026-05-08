package auth

import "crypto/subtle"

type User struct {
	ID           string
	ClientID     string
	ClientSecret string
}

type UserStore struct {
	usersByID         map[string]User
	userIDsByClientID map[string]string
}

func NewUserStore(users []User) *UserStore {
	store := &UserStore{
		usersByID:         make(map[string]User, len(users)),
		userIDsByClientID: make(map[string]string, len(users)),
	}

	for _, user := range users {
		store.usersByID[user.ID] = user
		store.userIDsByClientID[user.ClientID] = user.ID
	}

	return store
}

func (s *UserStore) FindByClientCredentials(clientID, clientSecret string) (User, bool) {
	userID, ok := s.userIDsByClientID[clientID]
	if !ok {
		return User{}, false
	}

	user, ok := s.usersByID[userID]
	if !ok {
		return User{}, false
	}

	if subtle.ConstantTimeCompare([]byte(clientSecret), []byte(user.ClientSecret)) != 1 {
		return User{}, false
	}

	return user, true
}

func defaultUsers() []User {
	return []User{
		{
			ID:           "user_1",
			ClientID:     "hello-world-client",
			ClientSecret: "hello-world-secret",
		},
	}
}
