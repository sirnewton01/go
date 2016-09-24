// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import "bufio"
import "errors"
import "os"
import "path/filepath"
import "strings"
import "strconv"

// If the ifindex is zero, interfaceTable returns mappings of all
// network interfaces. Otherwise it returns a mapping of a specific
// interface.
func interfaceTable(ifindex int) ([]Interface, error) {
	if ifindex == 0 {
		numIfc, err := getInterfaceCount()
		if err != nil {
			return nil, err
		}
		interfaces := make([]Interface, numIfc)
		for idx := 0; idx < numIfc; idx++ {
			ifc, err := readInterface(idx)
			if err != nil {
				return nil, err
			}
			interfaces[idx] = *ifc
		}

		return interfaces, nil
	}

	ifc, err := readInterface(ifindex)
	if err != nil {
		return nil, err
	}
	return []Interface{*ifc}, nil
}

func readInterface(id int) (*Interface, error) {
	iface := &Interface{}
	iface.Index = id + 1          // Offset the index by one to suit the contract
	iface.Name = strconv.Itoa(id) // Name however, is the ifc index in plan9

	ifaceStatus, err := os.Open(filepath.Join(netdir, "ipifc", iface.Name, "status"))
	if err != nil {
		return nil, err
	}
	defer ifaceStatus.Close()

	scanner := bufio.NewScanner(ifaceStatus)
	scanner.Scan()
	status := scanner.Text()
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	statusData := strings.Split(status, " ")
	if len(statusData) < 4 {
		return nil, errors.New("Invalid status file of interface: " + ifaceStatus.Name())
	}
	device := statusData[1]
	mtuStr := statusData[3]

	mtu, err := strconv.ParseInt(mtuStr, 10, 64)
	if err != nil {
		return nil, errors.New("Invalid status file of interface: " + ifaceStatus.Name())
	}
	iface.MTU = int(mtu)

	deviceAddrFile, err := os.Open(filepath.Join(device, "addr"))
	if err != nil {
		return nil, err
	}
	defer deviceAddrFile.Close()
	scanner = bufio.NewScanner(deviceAddrFile)
	scanner.Scan()
	address := scanner.Text()
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	if len(address) != 12 {
		return nil, errors.New("Interface has invalid hardware address")
	}
	address = address[0:1] + address[1:2] + ":" + address[2:3] + address[3:4] + ":" + address[4:5] + address[5:6] + ":" +
		address[6:7] + address[7:8] + ":" + address[8:9] + address[9:10] + ":" + address[10:11] + address[11:12]

	iface.HardwareAddr, err = ParseMAC(address)
	if err != nil {
		return nil, err
	}

	iface.Flags = FlagUp | FlagBroadcast | FlagLoopback

	return iface, nil
}

func getInterfaceCount() (int, error) {
	ipifc, err := os.Open(filepath.Join(netdir, "ipifc"))
	if err != nil {
		return -1, err
	}
	defer ipifc.Close()

	names, err := ipifc.Readdirnames(-1)
	if err != nil {
		return -1, err
	}

	count := 0
	for {
		found := false
		for _, name := range names {
			if name == strconv.Itoa(count) {
				found = true
				count++
				break
			}
		}
		if !found {
			break
		}
	}

	return count, nil
}

// If the ifi is nil, interfaceAddrTable returns addresses for all
// network interfaces. Otherwise it returns addresses for a specific
// interface.
func interfaceAddrTable(ifi *Interface) ([]Addr, error) {
	ifaces := []Interface{}
	if ifi == nil {
		var err error
		ifaces, err = interfaceTable(0)
		if err != nil {
			return nil, err
		}
	} else {
		ifaces = []Interface{*ifi}
	}

	addresses := make([]Addr, len(ifaces))
	for idx, iface := range ifaces {
		statusFile, err := os.Open(filepath.Join(netdir, "ipifc", iface.Name, "status"))
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(statusFile)
		scanner.Scan()
		scanner.Scan()
		err = scanner.Err()
		if err != nil {
			return nil, err
		}
		// This assumes only a single address for the interface
		ipline := scanner.Text()
		if ipline[0:1] != "\t" {
			return nil, errors.New("Cannot parse IP address for interface")
		}
		ipaddr := strings.Split(strings.Split(ipline, "\t")[1], " ")[0]

		ip := ParseIP(ipaddr)

		addr := IPAddr{IP: ip, Zone: ""}
		if addr.IP == nil {
			return nil, errors.New("Unable to parse IP address for interface")
		}
		addresses[idx] = &addr
	}

	return addresses, nil
}

// interfaceMulticastAddrTable returns addresses for a specific
// interface.
func interfaceMulticastAddrTable(ifi *Interface) ([]Addr, error) {
	return nil, nil
}
