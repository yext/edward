package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

func checkNotSudo() error {
	user, err := user.Current()
	if err != nil {
		return errors.WithStack(err)
	}
	if user.Uid == "0" {
		return errors.New("edward should not be run with sudo")
	}
	return nil
}

func ensureSudoAble() error {
	fmt.Println("One or more services use sudo. You may be prompted for your password.")
	var buffer bytes.Buffer

	buffer.WriteString("#!/bin/bash\n")
	buffer.WriteString("sudo echo Test > /dev/null\n")
	buffer.WriteString("ISCHILD=YES ")
	buffer.WriteString(strings.Join(os.Args, " "))
	buffer.WriteString("\n")

	log.Printf("Writing sudoAbility script\n")
	file, err := createScriptFile("sudoAbility", buffer.String())
	if err != nil {
		return errors.WithStack(err)
	}

	log.Printf("Launching sudoAbility script: %v\n", file.Name())
	err = syscall.Exec(file.Name(), []string{file.Name()}, os.Environ())
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func prepareForSudo() error {
	err := checkNotSudo()
	if err != nil {
		return errors.WithStack(err)
	}

	isChild := os.Getenv("ISCHILD")
	if isChild == "" {
		return errors.WithStack(ensureSudoAble())
	}
	log.Println("Child process, sudo should be available")
	return nil
}

func sudoIfNeeded(sgs []services.ServiceOrGroup) error {
	for _, sg := range sgs {
		if sg.IsSudo(services.OperationConfig{}) {
			log.Printf("sudo required for %v\n", sg.GetName())
			return errors.WithStack(prepareForSudo())
		}
	}
	log.Printf("sudo not required for any services/groups\n")
	return nil
}

func createScriptFile(suffix string, content string) (*os.File, error) {
	file, err := ioutil.TempFile(os.TempDir(), suffix)
	if err != nil {
		return nil, err
	}
	file.WriteString(content)
	file.Close()

	err = os.Chmod(file.Name(), 0777)
	if err != nil {
		return nil, err
	}

	return file, nil
}
