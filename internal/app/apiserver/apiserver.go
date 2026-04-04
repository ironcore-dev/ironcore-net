// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"

	apinetopenapi "github.com/ironcore-dev/ironcore-net/client-go/openapi"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/component-base/compatibility"

	"time"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	informers "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions"
	clientset "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	"github.com/ironcore-dev/ironcore-net/internal/apiserver"
	"github.com/ironcore-dev/ironcore-net/internal/registry/ipallocator"
	netflag "github.com/ironcore-dev/ironcore-net/utils/flag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
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

	// TODO: Remove this migration logic once all IPs have been migrated from OwnerReference to ephemeral label.
	// This is a one-time migration: once all IPs carry the ephemeral label, this hook becomes a no-op.
	// Check logs for "IP OwnerRef to ephemeral label migration: COMPLETE" to confirm migration is done.
	// After migration is complete, also remove the legacy OwnerReference check in ipallocator.Release().
	server.GenericAPIServer.AddPostStartHookOrDie("migrate-ip-ownerref-to-ephemeral-label", func(hookContext genericapiserver.PostStartHookContext) error {
		ironcoreNetClient, err := clientset.NewForConfig(config.GenericConfig.LoopbackClientConfig)
		if err != nil {
			klog.ErrorS(err, "Failed to create client for IP OwnerRef migration")
			return nil
		}

		const ipListPageSize = 500
		var allIPs []v1alpha1.IP
		continueToken := ""

		for {
			listOptions := metav1.ListOptions{
				Limit:    ipListPageSize,
				Continue: continueToken,
			}

			ipList, err := ironcoreNetClient.CoreV1alpha1().IPs("").List(hookContext, listOptions)
			if err != nil {
				klog.ErrorS(err, "Failed to list IPs for OwnerRef migration")
				return nil
			}

			allIPs = append(allIPs, ipList.Items...)

			continueToken = ipList.Continue
			if continueToken == "" {
				break
			}
		}

		ipList := &v1alpha1.IPList{
			Items: allIPs,
		}

		var migrated, skipped, failed int
		for i := range ipList.Items {
			select {
			case <-hookContext.Done():
				klog.InfoS("IP OwnerRef to ephemeral label migration: INTERRUPTED - Migration interrupted by server shutdown",
					"migrated", migrated, "skipped", skipped, "failed", failed)
				return nil
			default:
			}

			ip := &ipList.Items[i]

			if ip.Labels[ipallocator.IPEphemeralLabel] == "true" {
				skipped++
				continue
			}

			if metav1.GetControllerOf(ip) == nil {
				skipped++
				continue
			}

			if err := migrateIP(hookContext, ironcoreNetClient, ip); err != nil {
				klog.ErrorS(err, "Failed to migrate IP from OwnerRef to ephemeral label", "ip", klog.KObj(ip))
				failed++
				continue
			}
			migrated++
		}

		if migrated == 0 && failed == 0 {
			klog.InfoS("IP OwnerRef to ephemeral label migration: COMPLETE - No IPs with OwnerReference found, migration not needed",
				"totalIPs", len(ipList.Items))
			return nil
		}

		if failed == 0 && migrated > 0 {
			klog.InfoS("IP OwnerRef to ephemeral label migration: COMPLETE - All IPs successfully migrated",
				"migrated", migrated, "skipped", skipped, "total", len(ipList.Items))
		} else if failed > 0 {
			klog.InfoS("IP OwnerRef to ephemeral label migration: PARTIAL - Some IPs failed to migrate, will retry on next startup",
				"migrated", migrated, "failed", failed, "skipped", skipped, "total", len(ipList.Items))
		}
		return nil
	})

	return server.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

// TODO: Remove this migration logic once all IPs have been migrated from OwnerReference to ephemeral label.
// migrateIP migrates a single IP from OwnerReference-based detection to label-based detection.
// It uses retry logic with exponential backoff to handle transient API errors.
func migrateIP(ctx context.Context, ironcoreNetClient clientset.Interface, ip *v1alpha1.IP) error {
	patchObj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{
				ipallocator.IPEphemeralLabel: "true",
			},
			"ownerReferences": nil,
		},
	}

	patch, err := json.Marshal(patchObj)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	retryBackoff := wait.Backoff{
		Steps:    3,
		Duration: 10 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}
	return retry.OnError(
		retryBackoff,
		func(err error) bool {
			return apierrors.IsServerTimeout(err) ||
				apierrors.IsTimeout(err) ||
				apierrors.IsTooManyRequests(err) ||
				apierrors.IsInternalError(err) ||
				apierrors.IsServiceUnavailable(err)
		},
		func() error {
			patched, err := ironcoreNetClient.CoreV1alpha1().IPs(ip.Namespace).Patch(
				ctx,
				ip.Name,
				types.MergePatchType,
				patch,
				metav1.PatchOptions{},
			)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}

			if patched.Labels[ipallocator.IPEphemeralLabel] != "true" {
				return fmt.Errorf("patch succeeded but ephemeral label not found on IP %s", klog.KObj(patched))
			}

			if len(patched.OwnerReferences) != 0 {
				return fmt.Errorf("failed to remove ownerReferences from IP %s: %d ownerReferences still present", klog.KObj(patched), len(patched.OwnerReferences))
			}

			return nil
		},
	)
}
