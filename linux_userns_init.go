// +build linux

package libcontainer

import (
	"syscall"

	"github.com/docker/libcontainer/apparmor"
	"github.com/docker/libcontainer/configs"
	"github.com/docker/libcontainer/label"
	"github.com/docker/libcontainer/system"
)

type linuxUsernsInit struct {
	config *initConfig
}

func (l *linuxUsernsInit) Init() error {
	// join any namespaces via a path to the namespace fd if provided
	if err := joinExistingNamespaces(l.config.Config.Namespaces); err != nil {
		return newSystemError(err)
	}
	consolePath := l.config.Config.Console
	if consolePath != "" {
		// We use the containerConsolePath here, because the console has already been
		// setup by the side car process for the user namespace scenario.
		console := newConsoleFromPath(consolePath)
		if err := console.dupStdio(); err != nil {
			return newSystemError(err)
		}
	}
	if _, err := syscall.Setsid(); err != nil {
		return newSystemError(err)
	}
	if consolePath != "" {
		if err := system.Setctty(); err != nil {
			return newSystemError(err)
		}
	}
	if l.config.Cwd == "" {
		l.config.Cwd = "/"
	}
	if err := setupRlimits(l.config.Config); err != nil {
		return newSystemError(err)
	}
	// InitializeMountNamespace() can be executed only for a new mount namespace
	if l.config.Config.Namespaces.Contains(configs.NEWNS) {
		if err := setupRootfs(l.config.Config); err != nil {
			return newSystemError(err)
		}
	}
	if hostname := l.config.Config.Hostname; hostname != "" {
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			return newSystemError(err)
		}
	}
	if err := apparmor.ApplyProfile(l.config.Config.AppArmorProfile); err != nil {
		return newSystemError(err)
	}
	if err := label.SetProcessLabel(l.config.Config.ProcessLabel); err != nil {
		return newSystemError(err)
	}
	for _, path := range l.config.Config.ReadonlyPaths {
		if err := remountReadonly(path); err != nil {
			return newSystemError(err)
		}
	}
	for _, path := range l.config.Config.MaskPaths {
		if err := maskFile(path); err != nil {
			return newSystemError(err)
		}
	}
	pdeath, err := system.GetParentDeathSignal()
	if err != nil {
		return newSystemError(err)
	}
	if err := finalizeNamespace(l.config); err != nil {
		return newSystemError(err)
	}
	// finalizeNamespace can change user/group which clears the parent death
	// signal, so we restore it here.
	if err := pdeath.Restore(); err != nil {
		return newSystemError(err)
	}
	// Signal self if parent is already dead. Does nothing if running in a new
	// PID namespace, as Getppid will always return 0.
	if syscall.Getppid() == 1 {
		return syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
	}
	return system.Execv(l.config.Args[0], l.config.Args[0:], l.config.Env)
}
