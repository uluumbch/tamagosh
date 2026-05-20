package sftp

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/Candratama/tamagosh/internal/config"
)

type Entry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

type Client struct {
	mu     sync.RWMutex
	ssh    *ssh.Client
	sftp   *sftp.Client
	done   chan struct{}
	lostCh chan struct{}

	// dial parameters retained for reconnect.
	addr string
	cfg  *ssh.ClientConfig
}

// Auth describes how to authenticate the SFTP session.
type Auth struct {
	Method     string // "password" | "key"
	Password   string
	KeyPath    string
	Passphrase string
}

func Connect(c config.Connection, auth Auth) (*Client, error) {
	port := c.Port
	if port == 0 {
		port = 22
	}

	var methods []ssh.AuthMethod
	switch auth.Method {
	case "key":
		keyPath := auth.KeyPath
		// Defensive tilde expansion for legacy stored paths. Form.Build
		// now expands at save time, but old connections.json files may
		// still contain "~/..." entries. Go's os.ReadFile doesn't expand.
		if strings.HasPrefix(keyPath, "~/") {
			if home, herr := os.UserHomeDir(); herr == nil {
				keyPath = filepath.Join(home, keyPath[2:])
			}
		} else if keyPath == "~" {
			if home, herr := os.UserHomeDir(); herr == nil {
				keyPath = home
			}
		}
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}
		var signer ssh.Signer
		if auth.Passphrase == "" {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(auth.Passphrase))
		}
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		methods = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		methods = []ssh.AuthMethod{ssh.Password(auth.Password)}
	}

	home, _ := os.UserHomeDir()
	khPath := filepath.Join(home, ".ssh", "known_hosts")
	hkCb, err := hostKeyCallback(khPath)
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            methods,
		HostKeyCallback: hkCb,
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.Host, port)
	sc, fc, err := dial(addr, cfg)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		ssh:    sc,
		sftp:   fc,
		done:   make(chan struct{}),
		lostCh: make(chan struct{}, 1),
		addr:   addr,
		cfg:    cfg,
	}
	go cl.keepalive()
	return cl, nil
}

// dial opens a TCP connection with OS-level keepalive enabled (so the
// kernel sends probes independent of Go scheduling), performs the SSH
// handshake, and opens an SFTP subsystem channel.
func dial(addr string, cfg *ssh.ClientConfig) (*ssh.Client, *sftp.Client, error) {
	d := &net.Dialer{
		Timeout:   cfg.Timeout,
		KeepAlive: 30 * time.Second,
	}
	tcp, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dial: %w", err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(tcp, addr, cfg)
	if err != nil {
		_ = tcp.Close()
		return nil, nil, fmt.Errorf("ssh handshake: %w", err)
	}
	sc := ssh.NewClient(sshConn, chans, reqs)
	fc, err := sftp.NewClient(sc)
	if err != nil {
		_ = sc.Close()
		return nil, nil, fmt.Errorf("sftp open: %w", err)
	}
	return sc, fc, nil
}

// LostCh receives a single signal each time the keepalive loop concludes
// the SSH connection has died (3 consecutive missed probes). UI layer
// listens and may trigger Reconnect.
func (c *Client) LostCh() <-chan struct{} { return c.lostCh }

// DoneCh closes when Close is called — lets a waitLost subscriber exit
// cleanly instead of leaking when the session ends.
func (c *Client) DoneCh() <-chan struct{} { return c.done }

// Reconnect closes the current SSH/SFTP session and re-establishes a new
// one using the stored dial parameters. Safe to call from a goroutine; the
// keepalive loop tolerates the swap via the RWMutex.
func (c *Client) Reconnect() error {
	c.mu.Lock()
	if c.sftp != nil {
		_ = c.sftp.Close()
	}
	if c.ssh != nil {
		_ = c.ssh.Close()
	}
	sc, fc, err := dial(c.addr, c.cfg)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	c.ssh = sc
	c.sftp = fc
	c.mu.Unlock()
	return nil
}

// keepalive sends an OpenSSH keepalive request every 20s so NAT/firewall
// idle timeouts don't silently drop the connection. After 3 consecutive
// missed probes (~60s) it signals the lostCh once and keeps looping —
// UI may call Reconnect() to swap in a fresh session under the lock.
func (c *Client) keepalive() {
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	misses := 0
	signaled := false
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			c.mu.RLock()
			sshc := c.ssh
			c.mu.RUnlock()
			if sshc == nil {
				continue
			}
			_, _, err := sshc.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				misses++
				if misses >= 3 && !signaled {
					select {
					case c.lostCh <- struct{}{}:
					default:
					}
					signaled = true
				}
				continue
			}
			misses = 0
			signaled = false
		}
	}
}

func (c *Client) Close() error {
	if c.done != nil {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
	}
	if c.sftp != nil {
		c.sftp.Close()
	}
	if c.ssh != nil {
		return c.ssh.Close()
	}
	return nil
}

func (c *Client) Home() (string, error) {
	return c.sftp.Getwd()
}

func (c *Client) List(dir string) ([]Entry, error) {
	infos, err := c.sftp.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		out = append(out, Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: fi.Size(), ModTime: fi.ModTime()})
	}
	return out, nil
}

func (c *Client) Delete(remotePath string) error {
	info, err := c.sftp.Stat(remotePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return c.sftp.RemoveDirectory(remotePath)
	}
	return c.sftp.Remove(remotePath)
}

func (c *Client) Download(remotePath, localPath string) error {
	return c.DownloadProgress(remotePath, localPath, nil)
}

func (c *Client) Upload(localPath, remotePath string) error {
	return c.UploadProgress(localPath, remotePath, nil)
}

// ErrCancelled is returned by a transfer when the caller's stop func reports true.
var ErrCancelled = fmt.Errorf("transfer cancelled")

func (c *Client) DownloadProgress(remotePath, localPath string, onBytes func(int64)) error {
	return c.DownloadCancellable(remotePath, localPath, onBytes, nil)
}

func (c *Client) UploadProgress(localPath, remotePath string, onBytes func(int64)) error {
	return c.UploadCancellable(localPath, remotePath, onBytes, nil)
}

func (c *Client) DownloadCancellable(remotePath, localPath string, onBytes func(int64), stop func() bool) error {
	src, err := c.sftp.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmpPath := localPath + ".part"
	dst, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	r := &progressReader{r: src, cb: onBytes, stop: stop}
	if _, err := io.Copy(dst, r); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		return err
	}
	committed = true
	return nil
}

func (c *Client) UploadCancellable(localPath, remotePath string, onBytes func(int64), stop func() bool) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmpPath := remotePath + ".part"
	dst, err := c.sftp.Create(tmpPath)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = c.sftp.Remove(tmpPath)
		}
	}()

	r := &progressReader{r: src, cb: onBytes, stop: stop}
	if _, err := io.Copy(dst, r); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	// best-effort: some SFTP servers reject overwrite-on-rename
	_ = c.sftp.Remove(remotePath)
	if err := c.sftp.Rename(tmpPath, remotePath); err != nil {
		return err
	}
	committed = true
	return nil
}

func (c *Client) RemoteSize(path string) (int64, error) {
	info, err := c.sftp.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (c *Client) Mkdir(path string) error {
	return c.sftp.Mkdir(path)
}

func (c *Client) Rename(oldPath, newPath string) error {
	return c.sftp.Rename(oldPath, newPath)
}

func (c *Client) Stat(path string) (Entry, error) {
	info, err := c.sftp.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: info.Name(), IsDir: info.IsDir(), Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (c *Client) Chmod(path string, mode os.FileMode) error {
	return c.sftp.Chmod(path, mode)
}

func (c *Client) Walk(root string, fn func(path string, isDir bool, size int64) error) error {
	return walkRemote(c.sftp, root, fn)
}

func walkRemote(sc *sftp.Client, dir string, fn func(string, bool, int64) error) error {
	info, err := sc.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fn(dir, false, info.Size())
	}
	if err := fn(dir, true, 0); err != nil {
		return err
	}
	entries, err := sc.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		sub := dir
		if !strings.HasSuffix(sub, "/") {
			sub += "/"
		}
		sub += e.Name()
		if e.IsDir() {
			if err := walkRemote(sc, sub, fn); err != nil {
				return err
			}
		} else {
			if err := fn(sub, false, e.Size()); err != nil {
				return err
			}
		}
	}
	return nil
}

type progressReader struct {
	r    io.Reader
	cb   func(int64)
	stop func() bool
}

func (p *progressReader) Read(b []byte) (int, error) {
	if p.stop != nil && p.stop() {
		return 0, ErrCancelled
	}
	n, err := p.r.Read(b)
	if n > 0 && p.cb != nil {
		p.cb(int64(n))
	}
	return n, err
}

func Parent(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	p = strings.TrimRight(p, "/")
	parent := path.Dir(p)
	if parent == "." || parent == "" {
		return "/"
	}
	return parent
}

func Join(dir, name string) string {
	dir = strings.TrimRight(dir, "/")
	name = strings.TrimLeft(name, "/")
	if dir == "" {
		return "/" + name
	}
	return dir + "/" + name
}
