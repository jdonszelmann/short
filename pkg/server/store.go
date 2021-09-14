package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
)

const userPrefix = "user_"
const aliasPrefix = "alias_"

func prefix(prefix string, key string) []byte {
	return []byte(fmt.Sprintf("%s%s", prefix, key))
}

type Store struct {
	db *badger.DB
}

func NewStore() (*Store, error) {
	db, err := badger.Open(badger.DefaultOptions("store.db"))
	if err != nil {
		return nil, err
	}
	return &Store{
		db,
	}, nil
}

func (s Store) Close() {
	err := s.db.Close()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

type User struct {
	Name     string
	Password []byte
	Admin    bool
	Aliases  []string
}

type Alias struct {
	Owner string
	Url   string
	Alias string
}

func (s *Store) CreateUser(user User) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&user)
		if err != nil {
			return err
		}

		return txn.Set(prefix(userPrefix, user.Name), b.Bytes())
	})
}

func (s *Store) GetUser(name string) (User, error) {
	var res User
	return res, s.db.View(func(txn *badger.Txn) error {
		entry, err := txn.Get(prefix(userPrefix, name))
		if err != nil {
			return err
		}
		return entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&res)
		})
	})
}

func (s *Store) CountUsers() (int, error) {
	res := 0
	return res, s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		for it.Seek([]byte(userPrefix)); it.ValidForPrefix([]byte(userPrefix)); it.Next() {
			res += 1
		}

		it.Close()

		return nil
	})
}

func (s Store) UpdateUser(user *User) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&user)
		if err != nil {
			return err
		}

		return txn.Set(prefix(userPrefix, user.Name), b.Bytes())
	})
}

func (s Store) GetAlias(alias string) (*Alias, error) {
	var res *Alias
	return res, s.db.View(func(txn *badger.Txn) error {
		entry, err := txn.Get(prefix(aliasPrefix, alias))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&res)
		})
	})
}

func (s Store) CreateAlias(alias Alias) error {
	err := s.AddAliasToUser(alias.Owner, alias.Alias)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&alias)
		if err != nil {
			return err
		}

		return txn.Set(prefix(aliasPrefix, alias.Alias), b.Bytes())
	})
}

func (s Store) AddAliasToUser(owner string, alias string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		entry, err := txn.Get(prefix(userPrefix, owner))
		if err != nil {
			return err
		}

		var user User
		err = entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&user)
		})
		if err != nil {
			return err
		}

		user.Aliases = append(user.Aliases, alias)

		var b bytes.Buffer
		err = json.NewEncoder(&b).Encode(&user)
		if err != nil {
			return err
		}

		return txn.Set(prefix(userPrefix, user.Name), b.Bytes())
	})
}

func (s Store) RmAliasFromUser(owner string, alias string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		entry, err := txn.Get(prefix(userPrefix, owner))
		if err != nil {
			return err
		}

		var user User
		err = entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&user)
		})
		if err != nil {
			return err
		}

		var toRemove []int
		for i, a := range user.Aliases {
			if a == alias {
				toRemove = append(toRemove, i)
			}
		}

		for _, r := range toRemove {
			user.Aliases = append(user.Aliases[:r], user.Aliases[r+1:]...)
		}

		var b bytes.Buffer
		err = json.NewEncoder(&b).Encode(&user)
		if err != nil {
			return err
		}

		return txn.Set(prefix(userPrefix, user.Name), b.Bytes())
	})
}

func (s Store) GetUserAliases(user *User) ([]Alias, error) {
	if user.Aliases == nil {
		return nil, nil
	}

	res := make([]Alias, len(user.Aliases))

	return res, s.db.View(func(txn *badger.Txn) error {
		for i, alias := range user.Aliases {
			entry, err := txn.Get(prefix(aliasPrefix, alias))
			if err == badger.ErrKeyNotFound {
				return nil
			}
			if err != nil {
				return err
			}
			err = entry.Value(func(val []byte) error {
				return json.NewDecoder(bytes.NewBuffer(val)).Decode(&res[i])
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s Store) RmAlias(alias *Alias) error {
	if err := s.RmAliasFromUser(alias.Owner, alias.Alias); err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(prefix(aliasPrefix, alias.Alias))
	})
}
