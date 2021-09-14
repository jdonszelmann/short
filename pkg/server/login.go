package server

import (
	"golang.org/x/crypto/bcrypt"
	"log"
)

type LoginManager struct {
	store *Store
}

func NewLoginManager(store *Store) (*LoginManager, error) {
	count, err := store.CountUsers()
	if err != nil {
		return nil, err
	}
	res := &LoginManager{
		store,
	}

	if count == 0 {
		u := User{
			Name: "admin",
			Password: []byte(RandSeq(20)),
			Admin: true,
		}
		err := res.CreateUser(u)
		if err != nil {
			return nil, err
		}
		log.Printf("created new user with name %s and password %s", u.Name, string(u.Password))
	}

	return res, nil
}

type SessionUser struct {
	Name string
}

func (lm LoginManager) LogIn(lu User) (SessionUser, error) {
	user, err := lm.store.GetUser(lu.Name)
	if err != nil {
		return SessionUser{}, err
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, lu.Password); err != nil {
		return SessionUser{}, err
	}

	return SessionUser{
		Name:  user.Name,
	}, nil
}

func (lm LoginManager) LoggedIn(su SessionUser) (*User, error) {
	user, err := lm.store.GetUser(su.Name)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (lm LoginManager) CreateUser(user User) error {
	var err error
	user.Password, err = bcrypt.GenerateFromPassword(user.Password, bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return lm.store.CreateUser(user)
}

func (lm LoginManager) ChangePassword(user User, password string) error {
	var err error
	user.Password, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return lm.store.CreateUser(user)
}