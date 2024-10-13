package task

import (
	"context"
	"io"
	"math"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type DockerResult struct {
	Action      string
	ContainerId string
	Result      string
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachSterr   bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		Image:         t.Image,
		RestartPolicy: t.RestartPolicy,
		Env:           t.Env,
	}
}

type Docker struct {
	Client *client.Client
	Config Config
}

func NewDocker(c *Config) (*Docker, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &Docker{
		Client: dc,
		Config: *c,
	}, nil
}

func (d *Docker) Run(ctx context.Context) (*DockerResult, error) {
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}

	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}
	r := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}
	cc := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}
	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		return nil, err
	}

	err = d.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	out, err := d.Client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return nil, err
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return &DockerResult{
		ContainerId: resp.ID,
		Action:      "start",
		Result:      "success",
	}, nil
}

func (d *Docker) Stop(ctx context.Context, id string) (*DockerResult, error) {
	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		return nil, err
	}

	err = d.Client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		return nil, err
	}

	return &DockerResult{
		ContainerId: id,
		Action:      "stop",
		Result:      "success",
	}, nil
}
