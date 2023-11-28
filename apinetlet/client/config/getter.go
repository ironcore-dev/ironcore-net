// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"os"

	ironcorenetv1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	networkingv1alpha1 "github.com/ironcore-dev/ironcore/api/networking/v1alpha1"
	utilcertificate "github.com/ironcore-dev/ironcore/utils/certificate"
	"github.com/ironcore-dev/ironcore/utils/client/config"
	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apiserver/pkg/server/egressselector"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("client").WithName("config")

var (
	Getter = config.NewGetterOrDie(config.GetterOptions{
		Name:       "apinetlet",
		SignerName: certificatesv1.KubeAPIServerClientSignerName,
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   networkingv1alpha1.NetworkPluginCommonName("ironcore-net"),
				Organization: []string{networkingv1alpha1.NetworkPluginsGroup},
			},
		},
		GetUsages:      utilcertificate.DefaultKubeAPIServerClientGetUsages,
		NetworkContext: egressselector.ControlPlane.AsNetworkContext(),
	})

	GetConfig      = Getter.GetConfig
	GetConfigOrDie = Getter.GetConfigOrDie
)

func NewAPINetGetter(namespace string) (*config.Getter, error) {
	return config.NewGetter(config.GetterOptions{
		Name:       "apinetlet",
		SignerName: certificatesv1.KubeAPIServerClientSignerName,
		Template: &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   ironcorenetv1alpha1.APINetletCommonName(namespace),
				Organization: []string{ironcorenetv1alpha1.APINetletsGroup},
			},
		},
		GetUsages:      utilcertificate.DefaultKubeAPIServerClientGetUsages,
		NetworkContext: egressselector.ControlPlane.AsNetworkContext(),
	})
}

func NewAPINetGetterOrDie(namespace string) *config.Getter {
	getter, err := NewAPINetGetter(namespace)
	if err != nil {
		log.Error(err, "Error creating getter")
		os.Exit(1)
	}
	return getter
}
