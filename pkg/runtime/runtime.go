package runtime

import (
	"os"
	"runtime"
)

type Runtime struct {
	GOOS   string
	GOARCH string
}

func New() *Runtime {
	return &Runtime{
		GOOS:   goos(),
		GOARCH: goarch(),
	}
}

func goos() string {
	if s := os.Getenv("CLIVM_GOOS"); s != "" {
		return s
	}
	return runtime.GOOS
}

func goarch() string {
	if s := os.Getenv("CLIVM_GOARCH"); s != "" {
		return s
	}
	return runtime.GOARCH
}

func GOOSList() []string {
	return []string{"aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "linux", "netbsd", "openbsd", "plan9", "solaris", "windows"}
}

func GOARCHList() []string {
	return []string{"386", "amd64", "arm", "arm64", "mips", "mips64", "mips64le", "mipsle", "ppc64", "ppc64le", "riscv64", "s390x"}
}
