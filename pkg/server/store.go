package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
	"net/textproto"
)

const userPrefix = "user_"
const aliasPrefix = "alias_"
const filePrefix = "file_"


func prefix(prefix string, key string) []byte {
	return []byte(fmt.Sprintf("%s%s", prefix, key))
}

type Store struct {
	db *badger.DB
}

func NewStore(location string) (*Store, error) {
	db, err := badger.Open(badger.DefaultOptions(location))
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
	Password []byte
	File string
}

type File struct {
	Data []byte
	Mime textproto.MIMEHeader
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
		err := s.RmAliasFiles(txn, alias.Alias)
		if err != nil {
			return err
		}

		return txn.Delete(prefix(aliasPrefix, alias.Alias))
	})
}

func (s Store) GetUsers() ([]User, error) {
	var res []User
	return res, s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		for it.Seek([]byte(userPrefix)); it.ValidForPrefix([]byte(userPrefix)); it.Next() {

			var user User
			err := it.Item().Value(func(val []byte) error {
				return json.NewDecoder(bytes.NewBuffer(val)).Decode(&user)
			})

			if err != nil {
				return err
			}

			res = append(res, user)
		}

		it.Close()

		return nil
	})
}

func (s Store) RmUser(name string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var user User
		entry, err := txn.Get(prefix(userPrefix, name))
		if err != nil {
			return err
		}

		err = entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&user)
		})
		if err != nil {
			return err
		}

		for _, alias := range user.Aliases {
			err = s.RmAliasFiles(txn, alias)
			if err != nil {
				return err
			}
			err = txn.Delete(prefix(aliasPrefix, alias))
			if err != nil {
				return err
			}
		}

		return txn.Delete(prefix(userPrefix, name))
	})
}

func (s Store) SetAdmin(name string, value bool) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var user User
		entry, err := txn.Get(prefix(userPrefix, name))
		if err != nil {
			return err
		}
		err = entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&user)
		})
		if err != nil {
			return err
		}

		user.Admin = value

		var b bytes.Buffer
		err = json.NewEncoder(&b).Encode(&user)
		if err != nil {
			return err
		}

		return txn.Set(prefix(userPrefix, user.Name), b.Bytes())
	})
}

func (s Store) CreateFile(identifier string, f File) error {
	return s.db.Update(func(txn *badger.Txn) error {
		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(&f)
		if err != nil {
			return err
		}

		return txn.Set(prefix(filePrefix, identifier), b.Bytes())
	})
}

func (s Store) GetFile(identifier string) (*File, error) {
	var res *File
	return res, s.db.View(func(txn *badger.Txn) error {
		entry, err := txn.Get(prefix(filePrefix, identifier))
		if err != nil {
			return err
		}
		return entry.Value(func(val []byte) error {
			return json.NewDecoder(bytes.NewBuffer(val)).Decode(&res)
		})
	})
}

func (s Store) RmAliasFiles(txn *badger.Txn, aliasName string) error {
	var alias *Alias
	entry, err := txn.Get(prefix(aliasPrefix, aliasName))
	if err == badger.ErrKeyNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	err = entry.Value(func(val []byte) error {
		return json.NewDecoder(bytes.NewBuffer(val)).Decode(&alias)
	})
	if err != nil {
		return err
	}

	return txn.Delete(prefix(filePrefix, alias.File))
}
