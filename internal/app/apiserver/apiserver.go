// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/netip"

	apinetopenapi "github.com/ironcore-dev/ironcore-net/client-go/openapi"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/component-base/compatibility"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions"
	clientset "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	v1alpha1client "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/internal/apiserver"
	netflag "github.com/ironcore-dev/ironcore-net/utils/flag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	netutils "k8s.io/utils/net"
)

const (
	defaultEtcdPathPrefix = "/registry/apinet.ironcore.dev"

	defaultMinVNI = 200
	defaultMaxVNI = (1 << 24) - 1
)

type IronCoreNetServerOptions struct {
	RecommendedOptions    *options.RecommendedOptions
	SharedInformerFactory informers.SharedInformerFactory
	MinVNI                int32
	MaxVNI                int32
	PublicPrefix          []netip.Prefix
}

func (o *IronCoreNetServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	fs.Int32Var(&o.MinVNI, "min-vni", o.MinVNI, "Minimum VNI to allocate")
	fs.Int32Var(&o.MaxVNI, "max-vni", o.MaxVNI, "Maximum VNI to allocate")
	netflag.IPPrefixesVar(fs, &o.PublicPrefix, "public-prefix", o.PublicPrefix, "Public prefixes to allocate from")
}

func NewIronCoreNetServerOptions() *IronCoreNetServerOptions {
	o := &IronCoreNetServerOptions{
		RecommendedOptions: options.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			apiserver.Codecs.LegacyCodec(v1alpha1.SchemeGroupVersion),
		),
		MinVNI: defaultMinVNI,
		MaxVNI: defaultMaxVNI,
	}
	o.RecommendedOptions.Etcd.StorageConfig.EncodeVersioner = runtime.NewMultiGroupVersioner(v1alpha1.SchemeGroupVersion, schema.GroupKind{Group: v1alpha1.GroupName})
	return o
}

func NewCommandStartIronCoreNetServer(ctx context.Context, defaults *IronCoreNetServerOptions) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch an ironcore-net API server",
		Long:  "Launch an ironcore-net API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.Run(ctx); err != nil {
				return err
			}
			return nil
		},
	}

	o.AddFlags(cmd.Flags())
	utilfeature.DefaultMutableFeatureGate.AddFlag(cmd.Flags())

	return cmd
}

func (o *IronCoreNetServerOptions) Validate(args []string) error {
	var errs []error
	errs = append(errs, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *IronCoreNetServerOptions) Complete() error {
	return nil
}

func (o *IronCoreNetServerOptions) Config() (*apiserver.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %w", err)
	}

	o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
		ironcoreAPINetClient, err := clientset.NewForConfig(c.LoopbackClientConfig)
		if err != nil {
			return nil, err
		}

		informerFactory := informers.NewSharedInformerFactory(ironcoreAPINetClient, c.LoopbackClientConfig.Timeout)
		o.SharedInformerFactory = informerFactory

		return nil, nil
	}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)

	serverConfig.EffectiveVersion = compatibility.NewEffectiveVersionFromString("1.0", "", "")

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(apinetopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "ironcore-net-api"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(apinetopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
	serverConfig.OpenAPIV3Config.Info.Title = "ironcore-net-api"
	serverConfig.OpenAPIV3Config.Info.Version = "0.1"

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig: apiserver.ExtraConfig{
			MinVNI:             o.MinVNI,
			MaxVNI:             o.MaxVNI,
			PublicPrefix:       o.PublicPrefix,
			VersionedInformers: o.SharedInformerFactory,
		},
	}

	return config, nil
}

func (o *IronCoreNetServerOptions) Run(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHookOrDie("start-ironcore-net-server-informers", func(hookContext genericapiserver.PostStartHookContext) error {
		config.GenericConfig.SharedInformerFactory.Start(hookContext.Done())
		o.SharedInformerFactory.Start(hookContext.Done())
		return nil
	})

	// TODO: This is temporary migration code to strip OwnerReferences from legacy ephemeral IPs.
	// Remove this hook once all clusters have been migrated.
	server.GenericAPIServer.AddPostStartHookOrDie("migrate-ephemeral-ip-owner-references", func(hookContext genericapiserver.PostStartHookContext) error {
		ipClient, err := v1alpha1client.NewForConfig(hookContext.LoopbackClientConfig)
		if err != nil {
			slog.Error("Failed to create client for IP migration", "error", err)
			return nil
		}

		migrateEphemeralIPOwnerReferences(ipClient)
		return nil
	})

	return server.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

func migrateEphemeralIPOwnerReferences(ipClient v1alpha1client.CoreV1alpha1Interface) {
	var (
		continueToken string
		migrated      int
	)

	for {
		ipList, err := ipClient.IPs("").List(context.Background(), metav1.ListOptions{
			Limit:    500,
			Continue: continueToken,
		})
		if err != nil {
			slog.Error("Failed to list IPs for migration", "error", err)
			return
		}

		for i := range ipList.Items {
			ip := &ipList.Items[i]
			if metav1.GetControllerOf(ip) == nil {
				continue
			}

			patch := map[string]any{
				"metadata": map[string]any{
					"ownerReferences": []any{},
					"labels": map[string]string{
						v1alpha1.IPEphemeralLabel: "true",
					},
				},
			}
			patchData, err := json.Marshal(patch)
			if err != nil {
				slog.Error("Failed to marshal migration patch", "ip", ip.Name, "namespace", ip.Namespace, "error", err)
				continue
			}

			_, err = ipClient.IPs(ip.Namespace).Patch(context.Background(), ip.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			if err != nil {
				slog.Error("Failed to migrate IP", "ip", ip.Name, "namespace", ip.Namespace, "error", err)
				continue
			}
			migrated++
		}

		continueToken = ipList.Continue
		if continueToken == "" {
			break
		}
	}

	if migrated > 0 {
		slog.Info("Migrated ephemeral IPs: stripped OwnerReferences and ensured label", "count", migrated)
	}
}
