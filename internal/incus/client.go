/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package incus

import (
	"context"
	"fmt"
	"net/http"
	"os"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// Client provides operations for creating and deleting Incus instances.
type Client interface {
	Connect(ctx context.Context) error
	CreateInstance(ctx context.Context, name, image string, cpus, memoryMiB, rootDiskSizeGiB int) error
	DeleteInstance(ctx context.Context, name string) error
	InstanceExists(ctx context.Context, name string) (bool, error)
	Close() error
}

// clientImpl implements Client using the Incus Go library.
type clientImpl struct {
	socketPath string
	server     incus.InstanceServer
}

// ClientOption configures the Incus client.
type ClientOption func(*clientImpl)

// WithSocketPath sets the socket path for connecting to Incus. Empty string uses default.
func WithSocketPath(path string) ClientOption {
	return func(c *clientImpl) {
		c.socketPath = path
	}
}

// NewClient creates a new Incus client.
func NewClient(opts ...ClientOption) Client {
	c := &clientImpl{
		socketPath: os.Getenv("INCUS_SOCKET"),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Connect establishes a connection to the Incus daemon.
func (c *clientImpl) Connect(ctx context.Context) error {
	if c.server != nil {
		return nil
	}
	args := &incus.ConnectionArgs{}
	if ctx != nil {
		args = &incus.ConnectionArgs{}
	}
	server, err := incus.ConnectIncusUnixWithContext(ctx, c.socketPath, args)
	if err != nil {
		return fmt.Errorf("failed to connect to Incus: %w", err)
	}
	c.server = server
	return nil
}

// CreateInstance creates a new Incus VM instance from an image.
func (c *clientImpl) CreateInstance(ctx context.Context, name, image string, cpus, memoryMiB, rootDiskSizeGiB int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}

	// Default to reasonable values if not specified
	if cpus < 1 {
		cpus = 2
	}
	if memoryMiB < 1 {
		memoryMiB = 2048
	}
	if image == "" {
		image = "images:ubuntu/24.04"
	}

	instancePut := api.InstancePut{
		Config: map[string]string{
			"limits.cpu":          fmt.Sprintf("%d", cpus),
			"limits.memory":       fmt.Sprintf("%dMiB", memoryMiB),
			"security.secureboot": "false",
		},
		Profiles: []string{"default"},
	}

	// Override root disk size if specified
	if rootDiskSizeGiB > 0 {
		instancePut.Devices = map[string]map[string]string{
			"root": {
				"type": "disk",
				"pool": "default",
				"path": "/",
				"size": fmt.Sprintf("%dGiB", rootDiskSizeGiB),
			},
		}
	}

	req := api.InstancesPost{
		Name:         name,
		Type:         api.InstanceTypeVM,
		InstancePut:  instancePut,
		Source: api.InstanceSource{
			Type:  "image",
			Alias: image,
		},
		Start: true,
	}

	op, err := c.server.CreateInstance(req)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed waiting for instance creation: %w", err)
	}

	return nil
}

// DeleteInstance deletes an Incus instance.
func (c *clientImpl) DeleteInstance(ctx context.Context, name string) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}

	op, err := c.server.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed waiting for instance deletion: %w", err)
	}

	return nil
}

// InstanceExists checks if an instance exists.
func (c *clientImpl) InstanceExists(ctx context.Context, name string) (bool, error) {
	if err := c.Connect(ctx); err != nil {
		return false, err
	}

	_, _, err := c.server.GetInstance(name)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Close closes the connection. The Incus client doesn't expose a close method,
// but we clear the reference for consistency.
func (c *clientImpl) Close() error {
	c.server = nil
	return nil
}
