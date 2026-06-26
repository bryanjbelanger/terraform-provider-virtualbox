package virtualbox

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Client wraps VBoxManage CLI commands.
type Client struct {
	VBoxManagePath string
}

// NewClient creates a new VirtualBox client with the given path to VBoxManage.
func NewClient(vboxmanagePath string) *Client {
	return &Client{
		VBoxManagePath: vboxmanagePath,
	}
}

// Run executes a VBoxManage command with the given arguments and returns the output.
func (c *Client) Run(args ...string) (string, error) {
	cmd := exec.Command(c.VBoxManagePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vboxmanage error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RunWithStdin executes a VBoxManage command with stdin input.
func (c *Client) RunWithStdin(stdin []byte, args ...string) (string, error) {
	cmd := exec.Command(c.VBoxManagePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(stdin)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vboxmanage error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}