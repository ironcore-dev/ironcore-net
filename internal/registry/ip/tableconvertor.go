// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ip

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type convertor struct{}

var (
	objectMetaSwaggerDoc = metav1.ObjectMeta{}.SwaggerDoc()

	headers = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: objectMetaSwaggerDoc["name"]},
		{Name: "Type", Type: "string", Description: "The type of the IP"},
		{Name: "IP", Type: "string", Description: "The allocated IP"},
		{Name: "ClaimRef", Type: "string", Description: "The claiming entity, if any"},
		{Name: "Age", Type: "string", Format: "date", Description: objectMetaSwaggerDoc["creationTimestamp"]},
	}
)

func newTableConvertor() *convertor {
	return &convertor{}
}

func formatClaimRef(claimRef core.IPClaimRef) string {
	gr := schema.GroupResource{Group: claimRef.Group, Resource: claimRef.Resource}
	return fmt.Sprintf("%s %s", gr, claimRef.Name)
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
		ip := obj.(*core.IP)

		cells = append(cells, name)
		cells = append(cells, ip.Spec.Type)
		cells = append(cells, ip.Spec.IP.String())
		if claimRef := ip.Spec.ClaimRef; claimRef != nil {
			cells = append(cells, formatClaimRef(*claimRef))
		} else {
			cells = append(cells, "<none>")
		}
		cells = append(cells, age)

		return cells, nil
	})
	return tab, err
}
