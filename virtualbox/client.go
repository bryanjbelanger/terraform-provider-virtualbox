package virtualbox

import (
	"bytes"
	"context"
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

// RunContext executes a VBoxManage command with the given arguments and context, and returns the output.
func (c *Client) RunContext(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.VBoxManagePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vboxmanage error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RunWithStdinContext executes a VBoxManage command with stdin input and context.
func (c *Client) RunWithStdinContext(ctx context.Context, stdin []byte, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.VBoxManagePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(stdin)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("vboxmanage error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}