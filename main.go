package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type ProgramConfig struct {
	host            string
	user            string
	identityFile    string
	databaseName    string
	outputDirectory string
	containerName   string
	postgresUser    string
	verbose         bool
}

var cfg *ProgramConfig

func init() {
	cfg = &ProgramConfig{}
	flag.StringVar(&cfg.host, "h", "", "Host and port to connect to")
	flag.StringVar(&cfg.user, "u", "", "Username")
	flag.StringVar(&cfg.identityFile, "i", "", "Identify file to authenticate with")
	flag.StringVar(&cfg.databaseName, "d", "", "Postgres database name to dump")
	flag.StringVar(&cfg.outputDirectory, "o", ".", "Output directory")
	flag.StringVar(&cfg.containerName, "n", "postgres", "Docker container name")
	flag.StringVar(&cfg.postgresUser, "U", "postgres", "User to connect to database with")
	flag.BoolVar(&cfg.verbose, "v", false, "Verbose")
	flag.Parse()

	if cfg.host == "" {
		log.Fatal("-h argument must be set")
	} else if cfg.user == "" {
		log.Fatal("-u argument must be set")
	} else if cfg.identityFile == "" {
		log.Fatal("-i argument must be set")
	} else if cfg.databaseName == "" {
		log.Fatal("-d argument must be set")
	}
}

func main() {
	file, err := os.Create(getOutputFile(cfg.outputDirectory))
	ifFatal("Failed to open output file: %v", err)

	key, err := parseKeyFile(cfg.identityFile)
	ifFatal("Failed to load identity file: %v", err)

	sshConfig := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			key,
		},
		User: cfg.user,
		HostKeyCallback: func(_ string, _ net.Addr, _ ssh.PublicKey) error {
			return nil // Don't validate public key
		},
	}

	if cfg.verbose {
		log.Print("Connecting via ssh")
	}

	client, err := ssh.Dial("tcp", cfg.host, sshConfig)
	ifFatal("Failed to connect: %v", err)
	session, err := createSession(client)
	ifFatal("Failed to create session: %v", err)

	defer session.Close()

	// Setup stdin/stdout
	stdout, err := session.StdoutPipe()
	ifFatal("Failed to connect stdout: %v", err)
	stderr, err := session.StderrPipe()
	ifFatal("Failed to connect stderr: %v", err)
	go io.Copy(os.Stderr, stderr)

	// Start output writer
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go copyStdoutToFile(file, stdout, wg)

	// Execute command
	cmd := getCommand(cfg)
	if cfg.verbose {
		log.Printf("Executing command: %q", cmd)
	}
	err = session.Run(cmd)
	ifFatal("pg_dump command failed: %v", err)

	wg.Wait()
	fmt.Print(file.Name())
}
