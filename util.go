package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

const (
	dumpCommand = "docker exec -it %s pg_dump -C -U %s -d %s"
	dateFormat  = "2006-01-02T150405"
)

func getOutputFile(directory string) string {
	return path.Join(directory, time.Now().Format(dateFormat))
}

func getCommand(cfg *ProgramConfig) string {
	return fmt.Sprintf(dumpCommand, cfg.containerName, cfg.postgresUser, cfg.databaseName)
}

func parseKeyFile(file string) (ssh.AuthMethod, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(key), nil
}

func createSession(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err = session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}

func copyStdoutToFile(file *os.File, stdout io.Reader, wg *sync.WaitGroup) {
	defer func() {
		file.Close()
		wg.Done()
	}()

	_, err := io.Copy(file, stdout)
	if err != nil {
		log.Fatalf("Failed to write to output file: %v", err)
	}
}

func ifFatal(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	}
}
