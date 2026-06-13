package cmd

import (
	"github.com/cyberspacesec/go-snir/pkg/runner"
)

// proxyStrategyFlag 实现 pflag.Value 接口
type proxyStrategyFlag struct {
	value *runner.ProxyStrategy
}

func (f *proxyStrategyFlag) String() string {
	if f.value == nil {
		return ""
	}
	return string(*f.value)
}

func (f *proxyStrategyFlag) Set(s string) error {
	switch s {
	case "round-robin", "":
		*f.value = runner.ProxyRoundRobin
	case "random":
		*f.value = runner.ProxyRandom
	case "sequential":
		*f.value = runner.ProxySequential
	default:
		*f.value = runner.ProxyRoundRobin
	}
	return nil
}

func (f *proxyStrategyFlag) Type() string {
	return "string"
}
