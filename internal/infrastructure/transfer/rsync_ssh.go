package transfer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/UFFeScience/akoflow/internal/domain"
)

// RsyncSSH uses ssh/rsync installed on the gateway. Endpoint URI is
// ssh://user@host/absolute/base/path. A key path or extra SSH options may be
// provided in endpoint configuration as identityFile and sshOptions.
type RsyncSSH struct{}

func (RsyncSSH) CanHandle(e domain.TransferEndpoint) bool { return strings.HasPrefix(e.URI, "ssh://") }
func sshTarget(e domain.TransferEndpoint, name string) (string, string, error) {
	uri, err := url.Parse(e.URI)
	if err != nil || uri.Scheme != "ssh" {
		return "", "", fmt.Errorf("invalid SSH endpoint")
	}
	if uri.Host == "" || uri.Path == "" {
		return "", "", fmt.Errorf("ssh endpoint requires absolute path")
	}
	host, base := uri.Host, filepath.Clean(uri.Path)
	if uri.User != nil {
		host = uri.User.Username() + "@" + host
	}
	if host == "" || !filepath.IsAbs(base) {
		return "", "", fmt.Errorf("ssh endpoint requires host and absolute base path")
	}
	key := filepath.Clean(filepath.FromSlash(name))
	if name != "" && (filepath.IsAbs(key) || key == ".." || strings.HasPrefix(key, ".."+string(filepath.Separator))) {
		return "", "", fmt.Errorf("SSH transfer path escapes endpoint")
	}
	full := filepath.Join(base, key)
	if full != base && !strings.HasPrefix(full, base+string(filepath.Separator)) {
		return "", "", fmt.Errorf("SSH transfer path escapes endpoint")
	}
	return host, full, nil
}
func sshArgs(e domain.TransferEndpoint) []string {
	args := []string{}
	if key := e.Configuration["identityFile"]; key != "" {
		args = append(args, "-i", key)
	}
	if options := e.Configuration["sshOptions"]; options != "" {
		args = append(args, strings.Fields(options)...)
	}
	if uri, err := url.Parse(e.URI); err == nil {
		query := uri.Query()
		if identity := query.Get("identityFile"); identity != "" {
			args = append(args, "-i", identity)
		}
		if knownHosts := query.Get("knownHostsFile"); knownHosts != "" {
			args = append(args, "-o", "UserKnownHostsFile="+knownHosts, "-o", "StrictHostKeyChecking=yes")
		}
		if proxy := query.Get("proxyCommand"); proxy != "" {
			args = append(args, "-o", "ProxyCommand="+proxy)
		}
	}
	return args
}
func (RsyncSSH) Exists(ctx context.Context, e domain.TransferEndpoint, name string) (bool, error) {
	host, path, err := sshTarget(e, name)
	if err != nil {
		return false, err
	}
	args := append(sshArgs(e), host, "test -f "+shell(path))
	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	message := strings.TrimSpace(string(output))
	if message != "" {
		return false, fmt.Errorf("check SSH location: %w: %s", err, message)
	}
	return false, fmt.Errorf("check SSH location: %w", err)
}
func (RsyncSSH) Open(ctx context.Context, e domain.TransferEndpoint, name string, offset int64) (io.ReadCloser, error) {
	host, path, err := sshTarget(e, name)
	if err != nil {
		return nil, err
	}
	command := "cat -- " + shell(path)
	if offset > 0 {
		command = fmt.Sprintf("tail -c +%d -- %s", offset+1, shell(path))
	}
	args := append(sshArgs(e), host, command)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	return readCloser{Reader: out, close: cmd.Wait}, nil
}
func (RsyncSSH) Put(ctx context.Context, e domain.TransferEndpoint, name string, input io.Reader, offset int64) error {
	tmp, err := os.CreateTemp("", "akoflow-rsync-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, input); err == nil {
		err = tmp.Close()
	}
	if err != nil {
		return err
	}
	host, path, err := sshTarget(e, name)
	if err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, "ssh", append(sshArgs(e), host, "mkdir -p -- "+shell(filepath.Dir(path)))...).Run(); err != nil {
		return fmt.Errorf("create SSH staging directory: %w", err)
	}
	sshCommand := "ssh"
	if extra := sshArgs(e); len(extra) > 0 {
		sshCommand += " " + strings.Join(extra, " ")
	}
	args := []string{"-a", "--partial", "-e", sshCommand, tmp.Name(), host + ":" + path}
	if offset > 0 {
		args = []string{"-a", "--append-verify", "--partial", "-e", sshCommand, tmp.Name(), host + ":" + path}
	}
	return exec.CommandContext(ctx, "rsync", args...).Run()
}
func (RsyncSSH) Commit(ctx context.Context, e domain.TransferEndpoint, partial, final string) error {
	host, p, err := sshTarget(e, partial)
	if err != nil {
		return err
	}
	_, f, err := sshTarget(e, final)
	if err != nil {
		return err
	}
	args := append(sshArgs(e), host, "mkdir -p -- "+shell(filepath.Dir(f))+" && mv -- "+shell(p)+" "+shell(f))
	return exec.CommandContext(ctx, "ssh", args...).Run()
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r readCloser) Close() error { return r.close() }
func shell(v string) string       { return "'" + strings.ReplaceAll(v, "'", "'\\\"'\\\"'") + "'" }
