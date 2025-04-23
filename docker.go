package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pbergman/logger"
)

type writer struct {
	l func(m interface{})
}

func (w writer) Write(b []byte) (int, error) {

	var s = len(b)

	if b[s-1] == '\n' {
		s = s - 1
	}

	w.l(string(b[:s]))

	return s, nil
}

func CheckImages(ctx context.Context, cli *client.Client, cnf *Config, logger *logger.Logger) error {

	exist, err := HasImage(ctx, cli, cnf)

	if err != nil {
		return err
	}

	if false == exist {
		logger.Debug(fmt.Sprintf("docker image '%s' does not exist, fetching", cnf.Container.Name))
		return PullImage(ctx, cli, cnf, logger)
	} else {
		logger.Debug(fmt.Sprintf("docker image '%s' found", cnf.Container.Name))
	}

	return nil
}

func PullImage(ctx context.Context, cli *client.Client, cnf *Config, logger *logger.Logger) error {

	resp, err := cli.ImagePull(ctx, cnf.Container.Name, image.PullOptions{})

	if err != nil {
		return err
	}

	defer resp.Close()

	var decoder = json.NewDecoder(resp)
	var message jsonmessage.JSONMessage
	var writer = &writer{logger.Debug}

	for {

		err := decoder.Decode(&message)

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err := message.Display(writer, false); err != nil {
			return err
		}
	}

	return nil
}

func HasImage(ctx context.Context, cli *client.Client, cnf *Config) (bool, error) {
	sum, err := cli.ImageList(ctx, image.ListOptions{})

	if err != nil {
		return false, err
	}

	var tag = cnf.Container.Name

	if false == strings.Contains(tag, ":") {
		tag += ":latest"
	}

	for i, c := 0, len(sum); i < c; i++ {
		for a, b := 0, len(sum[i].RepoTags); a < b; a++ {
			if tag == sum[i].RepoTags[a] {
				return true, nil
			}
		}
	}

	return false, nil
}

func BuildSatis(ctx context.Context, cli *client.Client, user *user.User, cnf *Config, logger *logger.Logger, repos ...string) error {

	if err := CheckImages(ctx, cli, cnf, logger); err != nil {
		return err
	}

	uid, err := strconv.Atoi(user.Uid)

	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(user.Gid)

	if err != nil {
		return err
	}

	logger.Debug("checking out directory")

	if err := checkDirectories(cnf, uid, gid); err != nil {
		return err
	}

	logger.Debug("dumping statis config")

	if err := writeSatisConfig(cnf, uid, gid); err != nil {
		return err
	}

	args := []string{
		"build",
		"--no-interaction",
		"/build/satis.json",
		"/build/out",
	}

	if len(repos) > 0 {
		args = append(args, repos...)
	}

	logger.Debug(fmt.Sprintf("creating container with image '%s' and args '%s'", cnf.Container.Name, strings.Join(args, " ")))

	created, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: cnf.Container.Name,
			User:  user.Uid + ":" + user.Gid,
			Cmd:   args,
			Env:   []string{"HOME=" + user.HomeDir},
			Tty:   false,
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{
				Type:   cnf.Container.LogType,
				Config: cnf.Container.LogArgs,
			},
			AutoRemove: cnf.Container.AutoRemove,
			Binds:      getBinds(cnf, user),
		},
		nil,
		nil,
		"",
	)

	if err != nil {
		return err
	}

	logger.Debug("starting container")

	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return err
	}

	logger.Debug("waiting for container to be finished")

	ok, failed := cli.ContainerWait(ctx, created.ID, container.WaitConditionNotRunning)

	select {
	case err := <-failed:
		if err != nil {
			return err
		}
	case <-ok:
	}

	logger.Debug("finished running container")

	return nil
}

func checkDirectories(cnf *Config, uid, gid int) error {

	var curr string
	var dirs = []string{
		cnf.Directories.Build,
		"out",
	}

	for i, c := 0, len(dirs); i < c; i++ {

		path := filepath.Join(curr, dirs[i])

		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return err
		}

		curr = path
	}

	return nil
}

func writeSatisConfig(cnf *Config, uid, gid int) error {
	fd, err := os.Create(filepath.Join(cnf.Directories.Build, "satis.json"))

	if err != nil {
		return err
	}

	defer fd.Close()

	if err := json.NewEncoder(fd).Encode(cnf.SatisConfig); err != nil {
		return err
	}

	return fd.Chown(uid, gid)
}

func getBinds(cnf *Config, user *user.User) []string {

	var binds = []string{}

	binds = append(binds, cnf.Directories.Ssh+":"+user.HomeDir+"/.ssh")
	binds = append(binds, cnf.Directories.Build+":/build")

	for _, dir := range []string{"group", "passwd", "shadow"} {
		binds = append(binds, "/etc/"+dir+":/etc/"+dir+":ro")

	}

	if cnf.Directories.Composer != "" {
		binds = append(binds, cnf.Directories.Composer+":/composer")
	}

	return binds
}
