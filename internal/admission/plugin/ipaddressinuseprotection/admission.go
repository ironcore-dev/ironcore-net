// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ipaddressinuseprotection

import (
	"context"
	"io"
	"sync"

	"github.com/ironcore-dev/ironcore-net/internal/apis/core"
	"github.com/ironcore-dev/ironcore-net/internal/ipaddress"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PluginName = "IPAddressInUseProtection"
)

func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		plugin := newPlugin()
		return plugin, nil
	})
}

type ipAddressInUseProtectionPlugin struct {
	initOnce sync.Once
	initErr  error

	*admission.Handler
}

var _ admission.Interface = &ipAddressInUseProtectionPlugin{}

func newPlugin() *ipAddressInUseProtectionPlugin {
	return &ipAddressInUseProtectionPlugin{
		Handler: admission.NewHandler(admission.Create),
	}
}

func (p *ipAddressInUseProtectionPlugin) init() error {
	p.initOnce.Do(func() {
		p.initErr = func() error {
			return nil
		}()
	})
	return p.initErr
}

func (p *ipAddressInUseProtectionPlugin) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	log := klog.FromContext(ctx)

	ipAddress, ok := a.GetObject().(*core.IPAddress)
	if !ok {
		return nil
	}

	modified := controllerutil.AddFinalizer(ipAddress, ipaddress.ProtectionFinalizer)
	if modified {
		log.V(4).Info("Added IP address protection finalizer")
	}

	return nil
}
