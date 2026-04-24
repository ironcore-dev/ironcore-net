// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ironcore-dev/ironcore-net/utils/origin"
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
)

const (
	pollingInterval      = 50 * time.Millisecond
	eventuallyTimeout    = 3 * time.Second
	consistentlyDuration = 1 * time.Second
)

func TestOriginMetadataMigration(t *testing.T) {
	SetDefaultConsistentlyPollingInterval(pollingInterval)
	SetDefaultEventuallyPollingInterval(pollingInterval)
	SetDefaultEventuallyTimeout(eventuallyTimeout)
	SetDefaultConsistentlyDuration(consistentlyDuration)

	RegisterFailHandler(Fail)

	RunSpecs(t, "Origin Metadata Migration Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.34.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	DeferCleanup(testEnv.Stop)

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = Describe("OriginMetadataMigration", func() {
	ns := SetupNamespace(&k8sClient)

	namespacedOrigin := &origin.Origin{
		Name:       "test-origin",
		Namespaced: true,
	}

	clusterScopedOrigin := &origin.Origin{
		Name:       "test-cluster-origin",
		Namespaced: false,
	}

	It("should migrate a namespaced origin from labels to annotations", func(ctx context.Context) {
		By("creating a ConfigMap with the old-style origin (name in label instead of annotation)")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "migrate-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
					namespacedOrigin.UIDLabelKey():       "source-uid",
					namespacedOrigin.NamespaceLabelKey(): "source-namespace",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the name label was removed and name annotation was set")
		migratedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), migratedCM)).To(Succeed())

		Expect(migratedCM.Labels).NotTo(HaveKey(namespacedOrigin.NameAnnotationKey()))
		Expect(migratedCM.Annotations).To(HaveKeyWithValue(namespacedOrigin.NameAnnotationKey(), "source-name"))
		Expect(migratedCM.Labels).To(HaveKeyWithValue(namespacedOrigin.UIDLabelKey(), "source-uid"))
		Expect(migratedCM.Labels).To(HaveKeyWithValue(namespacedOrigin.NamespaceLabelKey(), "source-namespace"))
	})

	It("should migrate a cluster-scoped origin from labels to annotations", func(ctx context.Context) {
		By("creating a ConfigMap with an old-style cluster-scoped origin")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "migrate-cluster-",
				Labels: map[string]string{
					clusterScopedOrigin.NameAnnotationKey(): "source-name",
					clusterScopedOrigin.UIDLabelKey():       "source-uid",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      clusterScopedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the name label was removed and name annotation was set")
		migratedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), migratedCM)).To(Succeed())

		Expect(migratedCM.Labels).NotTo(HaveKey(clusterScopedOrigin.NameAnnotationKey()))
		Expect(migratedCM.Annotations).To(HaveKeyWithValue(clusterScopedOrigin.NameAnnotationKey(), "source-name"))
		Expect(migratedCM.Labels).To(HaveKeyWithValue(clusterScopedOrigin.UIDLabelKey(), "source-uid"))
	})

	It("should not modify objects that are already migrated", func(ctx context.Context) {
		By("creating a ConfigMap with the new-style origin (name in annotation, not in labels)")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "already-migrated-",
				Labels: map[string]string{
					namespacedOrigin.UIDLabelKey():       "source-uid",
					namespacedOrigin.NamespaceLabelKey(): "source-namespace",
				},
				Annotations: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		By("recording the resource version before migration")
		resourceVersion := cm.ResourceVersion

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the object was not modified")
		afterCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), afterCM)).To(Succeed())
		Expect(afterCM.ResourceVersion).To(Equal(resourceVersion))
	})

	It("should migrate multiple objects in a single call", func(ctx context.Context) {
		By("creating multiple ConfigMaps with old-style origin")
		for i := 0; i < 3; i++ {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      fmt.Sprintf("multi-migrate-%d", i),
					Labels: map[string]string{
						namespacedOrigin.NameAnnotationKey(): fmt.Sprintf("source-%d", i),
						namespacedOrigin.UIDLabelKey():       fmt.Sprintf("uid-%d", i),
						namespacedOrigin.NamespaceLabelKey(): "source-ns",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
		}

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying all objects were migrated")
		for i := 0; i < 3; i++ {
			cm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Namespace: ns.Name,
				Name:      fmt.Sprintf("multi-migrate-%d", i),
			}, cm)).To(Succeed())

			Expect(cm.Labels).NotTo(HaveKey(namespacedOrigin.NameAnnotationKey()))
			Expect(cm.Annotations).To(HaveKeyWithValue(namespacedOrigin.NameAnnotationKey(), fmt.Sprintf("source-%d", i)))
			Expect(cm.Labels).To(HaveKeyWithValue(namespacedOrigin.UIDLabelKey(), fmt.Sprintf("uid-%d", i)))
		}
	})

	It("should be idempotent when run multiple times", func(ctx context.Context) {
		By("creating a ConfigMap with old-style origin")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "idempotent-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
					namespacedOrigin.UIDLabelKey():       "source-uid",
					namespacedOrigin.NamespaceLabelKey(): "source-ns",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}

		By("running the migration the first time")
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("recording the resource version after first migration")
		migratedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), migratedCM)).To(Succeed())
		resourceVersion := migratedCM.ResourceVersion

		By("running the migration a second time")
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the object was not modified on the second run")
		afterCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), afterCM)).To(Succeed())
		Expect(afterCM.ResourceVersion).To(Equal(resourceVersion))
	})

	It("should preserve the origin data so it can be read back via Origin.DataOf", func(ctx context.Context) {
		By("creating a ConfigMap with old-style origin")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "data-roundtrip-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "my-source",
					namespacedOrigin.UIDLabelKey():       "my-uid-123",
					namespacedOrigin.NamespaceLabelKey(): "my-source-ns",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("reading back the migrated object and verifying DataOf returns correct data")
		migratedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), migratedCM)).To(Succeed())

		data := namespacedOrigin.DataOf(migratedCM)
		Expect(data).NotTo(BeNil())
		Expect(data.Name).To(Equal("my-source"))
		Expect(data.UID).To(Equal(types.UID("my-uid-123")))
		Expect(data.Namespace).To(Equal("my-source-ns"))
	})

	It("should only migrate objects matching the origin selector, not unrelated objects", func(ctx context.Context) {
		By("creating a ConfigMap with old-style origin")
		originCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "with-origin-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
					namespacedOrigin.UIDLabelKey():       "source-uid",
					namespacedOrigin.NamespaceLabelKey(): "source-ns",
				},
			},
		}
		Expect(k8sClient.Create(ctx, originCM)).To(Succeed())

		By("creating a ConfigMap without any origin labels")
		plainCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "no-origin-",
				Labels: map[string]string{
					"unrelated": "label",
				},
			},
		}
		Expect(k8sClient.Create(ctx, plainCM)).To(Succeed())
		plainResourceVersion := plainCM.ResourceVersion

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the origin ConfigMap was migrated")
		migratedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(originCM), migratedCM)).To(Succeed())
		Expect(migratedCM.Annotations).To(HaveKeyWithValue(namespacedOrigin.NameAnnotationKey(), "source-name"))
		Expect(migratedCM.Labels).NotTo(HaveKey(namespacedOrigin.NameAnnotationKey()))

		By("verifying the plain ConfigMap was not touched")
		untouchedCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(plainCM), untouchedCM)).To(Succeed())
		Expect(untouchedCM.ResourceVersion).To(Equal(plainResourceVersion))
	})

	It("should handle objects missing the UID label gracefully", func(ctx context.Context) {
		By("creating a ConfigMap with only the name label (missing UID)")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "missing-uid-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())
		resourceVersion := cm.ResourceVersion

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the object was not modified since origin data was incomplete")
		afterCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), afterCM)).To(Succeed())
		Expect(afterCM.ResourceVersion).To(Equal(resourceVersion))
	})

	It("should handle a namespaced origin missing the namespace label gracefully", func(ctx context.Context) {
		By("creating a ConfigMap with name and UID labels but no namespace label")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    ns.Name,
				GenerateName: "missing-ns-label-",
				Labels: map[string]string{
					namespacedOrigin.NameAnnotationKey(): "source-name",
					namespacedOrigin.UIDLabelKey():       "source-uid",
				},
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())
		resourceVersion := cm.ResourceVersion

		By("running the migration")
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())

		By("verifying the object was not modified since namespace label is required for namespaced origins")
		afterCM := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), afterCM)).To(Succeed())
		Expect(afterCM.ResourceVersion).To(Equal(resourceVersion))
	})

	It("should succeed with no errors when there are no objects to migrate", func(ctx context.Context) {
		migration := &OriginMetadataMigration{
			Client:      k8sClient,
			Origin:      namespacedOrigin,
			Type:        &corev1.ConfigMap{},
			ListOptions: []client.ListOption{client.InNamespace(ns.Name)},
		}
		Expect(migration.Migrate(ctx)).To(Succeed())
	})
})
