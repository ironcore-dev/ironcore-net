// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package loadbalancer

import (
	"context"

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
		{Name: "Type", Type: "string", Description: "The type of the load balancer"},
		{Name: "Network", Type: "string", Description: "The network of the load balancer"},
		{Name: "IPs", Type: "string", Description: "The IPs of the load balancer"},
		{Name: "Age", Type: "string", Format: "date", Description: objectMetaSwaggerDoc["creationTimestamp"]},
	}
)

func newTableConvertor() *convertor {
	return &convertor{}
}

func formatIPs(ips []core.LoadBalancerIP) string {
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
		loadBalancer := obj.(*core.LoadBalancer)

		cells = append(cells, name)
		cells = append(cells, loadBalancer.Spec.Type)
		cells = append(cells, loadBalancer.Spec.NetworkRef.Name)
		cells = append(cells, formatIPs(loadBalancer.Spec.IPs))
		cells = append(cells, age)

		return cells, nil
	})
	return tab, err
}
