package server

import (
	"github.com/dgraph-io/badger"
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
		_, err := res.CreateUser(u)
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

func (lm LoginManager) CreateUser(user User) (bool, error) {
	_, err := lm.store.GetUser(user.Name)
	if err != nil && err != badger.ErrKeyNotFound {
		return false, err
	} else if err == nil {
		return true, nil
	}

	user.Password, err = bcrypt.GenerateFromPassword(user.Password, bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}

	return false, lm.store.CreateUser(user)
}

func (lm LoginManager) ChangePassword(user User, password string) error {
	var err error
	user.Password, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return lm.store.CreateUser(user)
}

func (lm LoginManager) SetAdmin(name string, value bool) error {
	return lm.store.SetAdmin(name, value)
}