package adapter

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
)

type linuxInterface struct {
	name    string
	file    *os.File
	address string
	mtu     int
	isUp    bool
}

func newLinuxInterface(name string) (Interface, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/net/tun: %w", err)
	}

	ifreq, err := createIfreq(name)
	if err != nil {
		file.Close()
		return nil, err
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		file.Close()
		return nil, fmt.Errorf("failed to create TUN interface: %v", errno)
	}

	return &linuxInterface{
		name: name,
		file: file,
		mtu:  1500, // Default MTU
	}, nil
}

func createIfreq(name string) ([]byte, error) {
	// Create interface request structure (ifreq)
	var ifreq [40]byte
	copy(ifreq[:16], []byte(name))
	*(*uint16)(unsafe.Pointer(&ifreq[16])) = IFF_TUN | IFF_NO_PI
	return ifreq[:], nil
}

func (i *linuxInterface) Configure(cfg *Config) error {
	// Parse IP address and network to validate format
	if _, _, err := net.ParseCIDR(cfg.Address); err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	// Set IP address using ip command
	if err := exec.Command("ip", "addr", "add", cfg.Address, "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to set IP address: %w", err)
	}

	// Set MTU
	if err := exec.Command("ip", "link", "set", "mtu", fmt.Sprintf("%d", cfg.MTU), "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	// Bring interface up
	if err := exec.Command("ip", "link", "set", "up", "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	i.address = cfg.Address
	i.mtu = cfg.MTU
	i.isUp = true

	return nil
}

func (i *linuxInterface) Read(p []byte) (n int, err error) {
	return i.file.Read(p)
}

func (i *linuxInterface) Write(p []byte) (n int, err error) {
	return i.file.Write(p)
}

func (i *linuxInterface) Close() error {
	if i.file != nil {
		return i.file.Close()
	}
	return nil
}

func (i *linuxInterface) GetName() string {
	return i.name
}

func (i *linuxInterface) GetMTU() int {
	return i.mtu
}

func (i *linuxInterface) GetAddress() string {
	return i.address
}

func (i *linuxInterface) IsUp() bool {
	return i.isUp
}

func (i *linuxInterface) Cleanup() error {
	if i.isUp {
		// Bring interface down
		if err := exec.Command("ip", "link", "set", "down", "dev", i.name).Run(); err != nil {
			return fmt.Errorf("failed to bring interface down: %w", err)
		}

		// Remove IP address
		if err := exec.Command("ip", "addr", "del", i.address, "dev", i.name).Run(); err != nil {
			return fmt.Errorf("failed to remove IP address: %w", err)
		}

		i.isUp = false
	}

	return i.Close()
}
