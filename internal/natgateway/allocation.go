// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package natgateway

import (
	"github.com/onmetal/onmetal-api-net/apimachinery/api/net"
	"github.com/onmetal/onmetal-api-net/utils/container"
)

const (
	minEphemeralPort   int32 = 1024
	maxEphemeralPort   int32 = 65535
	noOfEphemeralPorts       = maxEphemeralPort + 1 - minEphemeralPort
)

type AllocationManager struct {
	portsPerNetworkInterface int32
	slots                    *container.KeySlots[net.IP]
}

func SlotsPerIP(portsPerNetworkInterface int32) int32 {
	return noOfEphemeralPorts / portsPerNetworkInterface
}

func NewAllocationManager(portsPerNetworkInterface int32, ips []net.IP) *AllocationManager {
	slotsPerIP := uint(noOfEphemeralPorts / portsPerNetworkInterface)
	slots := container.NewKeySlots(slotsPerIP, ips)

	return &AllocationManager{
		portsPerNetworkInterface: portsPerNetworkInterface,
		slots:                    slots,
	}
}

func (m *AllocationManager) HasIP(ip net.IP) bool {
	return m.slots.HasKey(ip)
}

func (m *AllocationManager) endPort(port int32) int32 {
	return port + m.portsPerNetworkInterface - 1
}

func (m *AllocationManager) slotForPorts(port, endPort int32) (uint, bool) {
	if port < minEphemeralPort || port >= endPort || endPort > maxEphemeralPort {
		return 0, false
	}
	if m.endPort(port) != endPort {
		return 0, false
	}
	return uint((port - minEphemeralPort) / m.portsPerNetworkInterface), true
}

func (m *AllocationManager) portsForSlot(slot uint) (port, endPort int32) {
	port = int32(slot)*m.portsPerNetworkInterface + minEphemeralPort
	endPort = m.endPort(port)
	return port, endPort
}

func (m *AllocationManager) Use(ip net.IP, port, endPort int32) bool {
	slot, ok := m.slotForPorts(port, endPort)
	if !ok {
		return false
	}

	return m.slots.Use(ip, slot)
}

func (m *AllocationManager) UseNextFree() (ip net.IP, port, endPort int32, ok bool) {
	ip, slot, ok := m.slots.UseNextFree()
	if !ok {
		return net.IP{}, 0, 0, false
	}

	port, endPort = m.portsForSlot(slot)
	return ip, port, endPort, true
}

func (m *AllocationManager) Total() int64 {
	return int64(m.slots.Total())
}

func (m *AllocationManager) Used() int64 {
	return int64(m.slots.Used())
}
