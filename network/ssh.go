// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/coreos/mantle/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/coreos/mantle/Godeps/_workspace/src/golang.org/x/crypto/ssh/agent"
)

const (
	defaultPort = 22
	defaultUser = "core"
	rsaKeySize  = 2048
)

// An interface for anything compatible with net.Dialer
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

// SSHAgent can manage keys, updates cloud config, and loves ponies.
// The embedded dialer is used for establishing new SSH connections.
type SSHAgent struct {
	agent.Agent
	Dialer
	User     string
	Socket   string
	sockDir  string
	listener *net.UnixListener
}

func NewSSHAgent(dialer Dialer) (*SSHAgent, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return nil, err
	}

	keyring := agent.NewKeyring()
	err = keyring.Add(key, nil, "core@default")
	if err != nil {
		return nil, err
	}

	sockDir, err := ioutil.TempDir("", "mantle-ssh-")
	if err != nil {
		return nil, err
	}

	// Use a similar naming scheme to ssh-agent
	sockPath := fmt.Sprintf("%s/agent.%d", sockDir, os.Getpid())
	sockAddr := &net.UnixAddr{Name: sockPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", sockAddr)
	if err != nil {
		os.RemoveAll(sockDir)
		return nil, err
	}

	a := &SSHAgent{
		Agent:    keyring,
		Dialer:   dialer,
		User:     defaultUser,
		Socket:   sockPath,
		sockDir:  sockDir,
		listener: listener,
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go agent.ServeAgent(a, conn)
		}
	}()

	return a, nil
}

func (a *SSHAgent) Close() error {
	a.listener.Close()
	return os.RemoveAll(a.sockDir)
}

// Add port to host if not already set.
func ensurePortSuffix(host string, port int) string {
	switch {
	case !strings.Contains(host, ":"):
		return fmt.Sprintf("%s:%d", host, port)
	case strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]"):
		return fmt.Sprintf("%s:%d", host, port)
	case strings.HasPrefix(host, "[") && strings.Contains(host, "]:"):
		return host
	case strings.Count(host, ":") > 1:
		return fmt.Sprintf("[%s]:%d", host, port)
	default:
		return host
	}
}

// Connect to the given host via SSH, the client will support
// agent forwarding but it must also be enabled per-session.
func (a *SSHAgent) NewClient(host string) (*ssh.Client, error) {
	sshcfg := ssh.ClientConfig{
		User: a.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(a.Signers),
		},
	}

	addr := ensurePortSuffix(host, defaultPort)
	tcpconn, err := a.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	sshconn, chans, reqs, err := ssh.NewClientConn(tcpconn, addr, &sshcfg)
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(sshconn, chans, reqs)
	err = agent.ForwardToAgent(client, a)
	if err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// Enable SSH Agent forwarding for the given session.
// This is just for convenience.
func (a *SSHAgent) RequestAgentForwarding(session *ssh.Session) error {
	return agent.RequestAgentForwarding(session)
}
