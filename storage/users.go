package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

type UserStorage map[string][]string

type UserListing struct {
	Users    map[string]struct{}
	AliasMap map[string]string
}

func (us *UserStorage) Read(path string) error {
	userFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer userFile.Close()

	dec := json.NewDecoder(userFile)
	err = dec.Decode(us)

	return err
}

func (us UserStorage) Write(path string) error {
	userFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer userFile.Close()

	enc := json.NewEncoder(userFile)
	err = enc.Encode(us)

	return err
}

func (us UserStorage) AddUser(user string, aliases []string) error {
	if _, ok := us[user]; ok {
		return fmt.Errorf("User %s already exists", user)
	}
	us[user] = aliases

	return nil
}

func (us UserStorage) AddAliases(user string, aliases []string) error {
	if _, ok := us[user]; !ok {
		return fmt.Errorf("No such user %s", user)
	}
	us[user] = append(us[user], aliases...)

	return nil
}

func (us UserStorage) String() string {
	var b bytes.Buffer

	w := tabwriter.NewWriter(&b, 4, 4, 0, '\t', 0)
	for user, aliases := range us {
		fmt.Fprintf(w, "%s:\t%v\n", user, aliases)
	}
	w.Flush()
	b.Truncate(b.Len() - 1)
	return b.String()
}

func (ul *UserListing) Parse(as UserStorage) {
	ul.Users = make(map[string]struct{})
	ul.AliasMap = make(map[string]string)

	for k, v := range as {
		ul.Users[k] = struct{}{}
		for _, alias := range v {
			ul.AliasMap[alias] = k
		}
	}
}

func ParseUserFile(path string) (UserListing, error) {
	var users UserListing
	var tmp UserStorage

	if path != "" {
		err := tmp.Read(path)
		if err != nil {
			return users, err
		}
	}

	users.Parse(tmp)

	return users, nil
}
