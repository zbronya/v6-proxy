package sysutils

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"
)

func AddV6Route(cidr string) {
	if !isRoot() {
		log.Fatal("You must run this program as root")
	}

	delCmd := exec.Command("ip", "route", "del", "local", cidr, "dev", "lo")
	delCmd.Run()

	addCmd := exec.Command("ip", "route", "add", "local", cidr, "dev", "lo")
	if err := addCmd.Run(); err != nil {
		log.Fatalf("Failed to add route: %v", err)
	} else {
		log.Printf("Added route %s dev lo", cidr)
	}

}

func SetV6Forwarding() {
	if !isRoot() {
		log.Fatal("You must run this program as root")
	}

	err := exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1").Run()
	if err != nil {
		log.Fatalf("Failed to enable IPv6 forwarding: %v", err)
	}
}

func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("Failed to get current user: %s\n", err)
		return false
	}
	return currentUser.Uid == "0"
}
