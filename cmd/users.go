package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/c-14/gtenlog/storage"
)

func userUsage() string {
	return `usage: gtenlog users [--help] <userFile> {add|addAlias|list} ...

Subcommands:
	add <username> [<aliasName>...]
	addAlias <username> <aliasName> [<aliasName>...]
	list`
}

func addUser(userFilePath string, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: grue users add <username> [<aliasName>...]")
	}

	var username string = args[0]
	var aliases []string = args[1:]

	var users storage.UserStorage = make(storage.UserStorage)
	err := users.Read(userFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Error opening user/alias mapping: %s", err)
	}

	err = users.AddUser(username, aliases)
	if err != nil {
		return err
	}

	return users.Write(userFilePath)
}

func addAlias(userFilePath string, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: grue users addAlias <username> <aliasName>...")
	}

	var username string = args[0]
	var aliases []string = args[1:]

	var users storage.UserStorage = make(storage.UserStorage)
	err := users.Read(userFilePath)
	if err != nil {
		return fmt.Errorf("Error opening user/alias mapping: %s", err)
	}

	err = users.AddAliases(username, aliases)
	if err != nil {
		return err
	}

	return users.Write(userFilePath)
}

func listUsers(userFilePath string, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: grue users list")
	}

	var users storage.UserStorage = make(storage.UserStorage)
	err := users.Read(userFilePath)
	if err != nil {
		return fmt.Errorf("Error opening user/alias mapping: %s", err)
	}

	fmt.Println(users)

	return nil
}

func Users(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(userUsage())
	}

	var err error
	switch arg := args[0]; arg {
	case "-h":
		fallthrough
	case "--help":
		return fmt.Errorf(userUsage())
	default:
		userFilePath := arg
		switch command := args[1]; command {
		case "add":
			fallthrough
		case "addUser":
			err = addUser(userFilePath, args[2:])
		case "addAlias":
			err = addAlias(userFilePath, args[2:])
		case "list":
			err = listUsers(userFilePath, args[2:])
		default:
			return fmt.Errorf(userUsage())
		}
	}
	return err
}
