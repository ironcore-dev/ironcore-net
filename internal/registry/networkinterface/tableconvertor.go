// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkinterface

import (
	"context"

	"github.com/ironcore-dev/ironcore-net/apimachinery/api/net"
	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	utilstrings "github.com/ironcore-dev/ironcore-net/utils/strings"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type convertor struct{}

var (
	objectMetaSwaggerDoc = metav1.ObjectMeta{}.SwaggerDoc()

	headers = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: objectMetaSwaggerDoc["name"]},
		{Name: "Network", Type: "string", Description: "The network of the network interface"},
		{Name: "IPs", Type: "string", Description: "The IPs of the network interface"},
		{Name: "PublicIPs", Type: "string", Description: "The public IPs of the network interface"},
		{Name: "Age", Type: "string", Format: "date", Description: objectMetaSwaggerDoc["creationTimestamp"]},
	}
)

func newTableConvertor() *convertor {
	return &convertor{}
}

func formatIPs(ips []net.IP) string {
	j := utilstrings.NewJoiner(",")
	for _, ip := range ips {
		j.Add(ip)
	}
	return j.String()
}

func formatPublicIPs(ips []core.NetworkInterfacePublicIP) string {
	j := utilstrings.NewJoiner(",")
	for _, ip := range ips {
		j.Add(ip.IP)
	}
	return j.String()
}

func (c *convertor) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	tab := &metav1.Table{
		ColumnDefinitions: headers,
	}

	if m, err := meta.ListAccessor(obj); err == nil {
		tab.ResourceVersion = m.GetResourceVersion()
		tab.Continue = m.GetContinue()
	} else {
		if m, err := meta.CommonAccessor(obj); err == nil {
			tab.ResourceVersion = m.GetResourceVersion()
		}
	}

	var err error
	tab.Rows, err = table.MetaToTableRow(obj, func(obj runtime.Object, m metav1.Object, name, age string) (cells []interface{}, err error) {
		nic := obj.(*core.NetworkInterface)

		cells = append(cells, name)
		cells = append(cells, nic.Spec.NetworkRef.Name)
		cells = append(cells, formatIPs(nic.Spec.IPs))
		cells = append(cells, formatPublicIPs(nic.Spec.PublicIPs))
		cells = append(cells, age)

		return cells, nil
	})
	return tab, err
}
