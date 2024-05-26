package sysutils

import (
	"github.com/zbronya/v6-proxy/log"
	"os/exec"
	"os/user"
)

func AddV6Route(cidr string) {
	if !isRoot() {
		log.GetLogger().Fatal("You must run this program as root")
	}

	delCmd := exec.Command("ip", "route", "del", "local", cidr, "dev", "lo")
	delCmd.Run()

	addCmd := exec.Command("ip", "route", "add", "local", cidr, "dev", "lo")
	if err := addCmd.Run(); err != nil {
		log.GetLogger().Fatal("Failed to add route: %v", err)
	} else {
		log.GetLogger().Info("Added route %s dev lo", cidr)
	}

}

func SetV6Forwarding() {
	if !isRoot() {
		log.GetLogger().Fatal("You must run this program as root")
	}

	err := exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1").Run()
	if err != nil {
		log.GetLogger().Fatal("Failed to enable IPv6 forwarding: %v", err)
	}
}

func SetIpNonLocalBind() {
	if !isRoot() {
		log.GetLogger().Fatal("You must run this program as root")
	}

	err := exec.Command("sysctl", "-w", "net.ipv6.ip_nonlocal_bind=1").Run()
	if err != nil {
		log.GetLogger().Fatal("Failed to enable IPv6 non local bind: %v", err)
	}

}

func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		log.GetLogger().Fatal("Failed to get current user: %s\n", err)
		return false
	}
	return currentUser.Uid == "0"
}
