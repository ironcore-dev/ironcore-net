// Copyright 2023 IronCore authors
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

package apiserver

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	apinetopenapi "github.com/ironcore-dev/ironcore-net/client-go/openapi"
	"k8s.io/apiserver/pkg/endpoints/openapi"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"github.com/ironcore-dev/ironcore-net/client-go/informers"
	clientset "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet"
	"github.com/ironcore-dev/ironcore-net/internal/apiserver"
	netflag "github.com/ironcore-dev/ironcore-net/utils/flag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/features"
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

type OnmetalAPINetServerOptions struct {
	RecommendedOptions    *options.RecommendedOptions
	SharedInformerFactory informers.SharedInformerFactory
	MinVNI                int32
	MaxVNI                int32
	PublicPrefix          []netip.Prefix
}

func (o *OnmetalAPINetServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.RecommendedOptions.AddFlags(fs)
	fs.Int32Var(&o.MinVNI, "min-vni", o.MinVNI, "Minimum VNI to allocate")
	fs.Int32Var(&o.MaxVNI, "max-vni", o.MaxVNI, "Maximum VNI to allocate")
	netflag.IPPrefixesVar(fs, &o.PublicPrefix, "public-prefix", o.PublicPrefix, "Public prefixes to allocate from")
}

func NewOnmetalAPINetServerOptions() *OnmetalAPINetServerOptions {
	o := &OnmetalAPINetServerOptions{
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

func NewCommandStartOnmetalAPINetServer(ctx context.Context, defaults *OnmetalAPINetServerOptions) *cobra.Command {
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

func (o *OnmetalAPINetServerOptions) Validate(args []string) error {
	var errs []error
	errs = append(errs, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errs)
}

func (o *OnmetalAPINetServerOptions) Complete() error {
	return nil
}

func (o *OnmetalAPINetServerOptions) Config() (*apiserver.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %w", err)
	}

	o.RecommendedOptions.Etcd.StorageConfig.Paging = utilfeature.DefaultFeatureGate.Enabled(features.APIListChunking)

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

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(apinetopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "ironcore-net"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	if utilfeature.DefaultFeatureGate.Enabled(features.OpenAPIV3) {
		serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIConfig(apinetopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
		serverConfig.OpenAPIV3Config.Info.Title = "ironcore-net"
		serverConfig.OpenAPIV3Config.Info.Version = "0.1"
	}

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

func (o *OnmetalAPINetServerOptions) Run(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHookOrDie("start-ironcore-net-server-informers", func(context genericapiserver.PostStartHookContext) error {
		config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
		o.SharedInformerFactory.Start(context.StopCh)
		return nil
	})

	return server.GenericAPIServer.PrepareRun().Run(ctx.Done())
}
