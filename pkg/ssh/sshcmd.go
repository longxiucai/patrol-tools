package ssh

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/longxiucai/patrol-tools/pkg/common"
	utilsnet "github.com/longxiucai/patrol-tools/pkg/util/net"

	"github.com/golang/glog"
)

const SUDO = "sudo "

func (s *SSH) Ping(host net.IP) error {
	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		return nil
	}
	client, _, err := s.Connect(host)
	if err != nil {
		return fmt.Errorf("[ssh %s] failed to create ssh session: %v", host, err)
	}
	err = client.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SSH) CmdAsync(host net.IP, cmds ...string) error {
	var execFunc func(cmd string) error

	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		execFunc = func(cmd string) error {
			c := exec.Command("/bin/sh", "-c", cmd)
			stdout, err := c.StdoutPipe()
			if err != nil {
				return err
			}

			stderr, err := c.StderrPipe()
			if err != nil {
				return err
			}

			if err := c.Start(); err != nil {
				return fmt.Errorf("failed to start command %s: %v", cmd, err)
			}

			ReadPipe(stdout, stderr, s.IsStdout)

			err = c.Wait()
			if err != nil {
				return fmt.Errorf("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			}
			return nil
		}
	} else {
		execFunc = func(cmd string) error {
			client, session, err := s.Connect(host)
			if err != nil {
				return fmt.Errorf("failed to create ssh session for %s: %v", host, err)
			}
			defer client.Close()
			defer session.Close()
			stdout, err := session.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed to create stdout pipe for %s: %v", host, err)
			}
			stderr, err := session.StderrPipe()
			if err != nil {
				return fmt.Errorf("failed to create stderr pipe for %s: %v", host, err)
			}

			if err := session.Start(cmd); err != nil {
				return fmt.Errorf("failed to start command %s on %s: %v", cmd, host, err)
			}

			ReadPipe(stdout, stderr, s.IsStdout)

			err = session.Wait()
			if err != nil {
				return fmt.Errorf("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			}

			return nil
		}
	}

	for _, cmd := range cmds {
		if cmd == "" {
			continue
		}
		if s.User != common.ROOT {
			cmd = fmt.Sprintf("sudo -E /bin/sh <<EOF\n%s\nEOF", cmd)
		}
		if err := execFunc(cmd); err != nil {
			glog.Infof("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			return err
		}
	}

	return nil
}

func (s *SSH) Cmd(host net.IP, cmd string) ([]byte, error) {
	if s.User != common.ROOT {
		cmd = fmt.Sprintf("sudo -E /bin/sh <<EOF\n%s\nEOF", cmd)
	}
	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		b, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
		if err != nil {
			glog.Infof("failed to execute command(%s) on host(%s): error(%v)", cmd, host, err)
			return nil, err
		}
		return b, err
	}

	client, session, err := s.Connect(host)
	if err != nil {
		return nil, fmt.Errorf("[ssh][%s] create ssh session failed, %s", host, err)
	}
	defer client.Close()
	defer session.Close()
	b, err := session.CombinedOutput(cmd)
	if err != nil {
		glog.Infof("[ssh][%s]run command failed [%s]", host, cmd)
		return b, fmt.Errorf("[ssh][%s]run command failed [%s]", host, cmd)
	}

	return b, nil
}

// CmdToString is in host exec cmd and replace to spilt str
func (s *SSH) CmdToString(host net.IP, cmd, split string) (string, error) {
	data, err := s.Cmd(host, cmd)
	str := string(data)
	if err != nil {
		return str, fmt.Errorf("failed to exec command(%s) on host(%s): %v", cmd, host, err)
	}
	if data != nil {
		str = strings.ReplaceAll(str, "\r\n", split)
		str = strings.ReplaceAll(str, "\n", split)
		return str, nil
	}
	return str, fmt.Errorf("command %s %s return nil", host, cmd)
}
