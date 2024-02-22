package shell

import (
	"fmt"

	"github.com/longxiucai/patrol-tools/pkg/common"
)

type SHELLCONFIG struct {
	Shell []SHELL `mapstructure:"shell-rules" yaml:"shell-rules"`
}

type SHELL struct {
	Name     string   `mapstructure:"name" yaml:"name"`
	Command  string   `mapstructure:"command" yaml:"command"`
	Operator string   `mapstructure:"operator" yaml:"operator"`
	Value    string   `mapstructure:"value" yaml:"value"`
	Selector []string `mapstructure:"node-selector" yaml:"node-selector"`
}

func (sc *SHELLCONFIG) Exec(ssh *common.SSHCONFIG) {
	for shell := range sc.Shell {
		fmt.Println(shell)
	}
}
