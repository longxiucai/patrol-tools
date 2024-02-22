package ssh

import (
	"fmt"
	"net"
	"time"

	"github.com/longxiucai/patrol-tools/pkg/common"

	"github.com/golang/glog"
	"github.com/imdario/mergo"

	netUtils "github.com/longxiucai/patrol-tools/pkg/util/net"
)

type Interface interface {
	// Copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	// Copy(host net.IP, srcFilePath, dstFilePath string) error
	// Fetch copy remote host files to localhost
	// Fetch(host net.IP, srcFilePath, dstFilePath string) error
	// CmdAsync exec command on remote host, and asynchronous return logs
	CmdAsync(host net.IP, cmd ...string) error
	// Cmd exec command on remote host, and return combined standard output and standard error
	Cmd(host net.IP, cmd string) ([]byte, error)
	// IsFileExist check remote file exist or not
	// IsFileExist(host net.IP, remoteFilePath string) (bool, error)
	// RemoteDirExist Remote file existence returns true, nil
	// RemoteDirExist(host net.IP, remoteDirpath string) (bool, error)
	// CmdToString exec command on remote host, and return spilt standard output and standard error
	CmdToString(host net.IP, cmd, spilt string) (string, error)
	// Platform Get remote platform
	// Platform(host net.IP) (v1.Platform, error)

	Ping(host net.IP) error
}

type SSH struct {
	IsStdout     bool
	Encrypted    bool
	User         string
	Password     string
	Port         string
	PkFile       string
	PkPassword   string
	Timeout      *time.Duration
	LocalAddress []net.Addr
	// Fs           fs.Interface
}

func NewSSHClient(ssh *common.SSH, isStdout bool) Interface {
	if ssh.User == "" {
		ssh.User = common.ROOT
	}
	address, err := netUtils.GetLocalHostAddresses()
	if err != nil {
		glog.Warningf("failed to get local address: %v", err)
	}
	return &SSH{
		IsStdout:     isStdout,
		Encrypted:    ssh.Encrypted,
		User:         ssh.User,
		Password:     ssh.Passwd,
		Port:         ssh.Port,
		PkFile:       ssh.Pk,
		PkPassword:   ssh.PkPasswd,
		LocalAddress: address,
		// Fs:           fs.NewFilesystem(),
	}
}

// GetHostSSHClient is used to executed bash command and no std out to be printed.
func GetHostSSHClient(hostIP net.IP, sshConfig *common.SSHCONFIG) (Interface, error) {
	for _, host := range sshConfig.Hosts {
		for _, ip := range host.IPS {
			if hostIP.Equal(ip) {
				if err := mergo.Merge(&host.SSH, &sshConfig.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&host.SSH, false), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to get host ssh client: host ip %s not in hosts ip list", hostIP)
}

// NewStdoutSSHClient is used to show std out when execute bash command.
func NewStdoutSSHClient(hostIP net.IP, sshConfig *common.SSHCONFIG) (Interface, error) {
	for _, host := range sshConfig.Hosts {
		for _, ip := range host.IPS {
			if hostIP.Equal(ip) {
				if err := mergo.Merge(&host.SSH, &sshConfig.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&host.SSH, true), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to get host ssh client: host ip %s not in hosts ip list", hostIP)
}
