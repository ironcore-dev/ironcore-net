// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package origin

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestOrigin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Origin Suite")
}

var _ = Describe("Origin", func() {
	var namespaced, clusterScoped *Origin

	BeforeEach(func() {
		namespaced = &Origin{Name: "test-origin", Namespaced: true}
		clusterScoped = &Origin{Name: "test-origin", Namespaced: false}
	})

	Describe("key helpers", func() {
		It("should produce correct label and annotation keys", func() {
			Expect(namespaced.UIDLabelKey()).To(Equal("test-origin-uid"))
			Expect(namespaced.NamespaceLabelKey()).To(Equal("test-origin-namespace"))
			Expect(namespaced.NameAnnotationKey()).To(Equal("test-origin-name"))
		})
	})

	Describe("Labels", func() {
		It("should include UID and namespace for a namespaced origin", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Namespace: "src-ns",
				Name:      "src-name",
				UID:       "src-uid",
			}}

			lbls := namespaced.Labels(source)
			Expect(lbls).To(HaveKeyWithValue("test-origin-uid", "src-uid"))
			Expect(lbls).To(HaveKeyWithValue("test-origin-namespace", "src-ns"))
		})

		It("should include UID but not namespace for a cluster-scoped origin", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: "src-name",
				UID:  "src-uid",
			}}

			lbls := clusterScoped.Labels(source)
			Expect(lbls).To(HaveKeyWithValue("test-origin-uid", "src-uid"))
			Expect(lbls).NotTo(HaveKey("test-origin-namespace"))
		})
	})

	Describe("Annotations", func() {
		It("should include the source name", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "my-source"}}

			anns := namespaced.Annotations(source)
			Expect(anns).To(HaveKeyWithValue("test-origin-name", "my-source"))
		})
	})

	Describe("SetOrigin", func() {
		It("should set labels and annotations on the target object", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Namespace: "src-ns",
				Name:      "src-name",
				UID:       "src-uid",
			}}
			target := &corev1.ConfigMap{}

			namespaced.SetOrigin(source, target)

			Expect(target.Labels).To(HaveKeyWithValue("test-origin-uid", "src-uid"))
			Expect(target.Labels).To(HaveKeyWithValue("test-origin-namespace", "src-ns"))
			Expect(target.Annotations).To(HaveKeyWithValue("test-origin-name", "src-name"))
		})

		It("should preserve existing labels and annotations on the target", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: "src-name",
				UID:  "src-uid",
			}}
			target := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{"existing": "label"},
				Annotations: map[string]string{"existing": "annotation"},
			}}

			clusterScoped.SetOrigin(source, target)

			Expect(target.Labels).To(HaveKeyWithValue("existing", "label"))
			Expect(target.Labels).To(HaveKeyWithValue("test-origin-uid", "src-uid"))
			Expect(target.Annotations).To(HaveKeyWithValue("existing", "annotation"))
			Expect(target.Annotations).To(HaveKeyWithValue("test-origin-name", "src-name"))
		})
	})

	Describe("RemoveOrigin", func() {
		It("should remove all origin labels and annotations", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid":       "uid",
					"test-origin-namespace": "ns",
					"unrelated":             "keep",
				},
				Annotations: map[string]string{
					"test-origin-name": "name",
					"unrelated":        "keep",
				},
			}}

			namespaced.RemoveOrigin(obj)

			Expect(obj.Labels).NotTo(HaveKey("test-origin-uid"))
			Expect(obj.Labels).NotTo(HaveKey("test-origin-namespace"))
			Expect(obj.Annotations).NotTo(HaveKey("test-origin-name"))
			Expect(obj.Labels).To(HaveKeyWithValue("unrelated", "keep"))
			Expect(obj.Annotations).To(HaveKeyWithValue("unrelated", "keep"))
		})

		It("should not panic on an object with nil labels and annotations", func() {
			obj := &corev1.ConfigMap{}
			Expect(func() { namespaced.RemoveOrigin(obj) }).NotTo(Panic())
		})
	})

	Describe("DataOf", func() {
		It("should return origin data from a properly annotated namespaced object", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid":       "the-uid",
					"test-origin-namespace": "the-ns",
				},
				Annotations: map[string]string{
					"test-origin-name": "the-name",
				},
			}}

			data := namespaced.DataOf(obj)
			Expect(data).NotTo(BeNil())
			Expect(data.Name).To(Equal("the-name"))
			Expect(data.UID).To(Equal(types.UID("the-uid")))
			Expect(data.Namespace).To(Equal("the-ns"))
		})

		It("should return origin data from a cluster-scoped object without namespace", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid": "the-uid",
				},
				Annotations: map[string]string{
					"test-origin-name": "the-name",
				},
			}}

			data := clusterScoped.DataOf(obj)
			Expect(data).NotTo(BeNil())
			Expect(data.Name).To(Equal("the-name"))
			Expect(data.UID).To(Equal(types.UID("the-uid")))
			Expect(data.Namespace).To(BeEmpty())
		})

		It("should return nil when the name annotation is missing", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid":       "uid",
					"test-origin-namespace": "ns",
				},
			}}

			Expect(namespaced.DataOf(obj)).To(BeNil())
		})

		It("should return nil when the UID label is missing", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-namespace": "ns",
				},
				Annotations: map[string]string{
					"test-origin-name": "name",
				},
			}}

			Expect(namespaced.DataOf(obj)).To(BeNil())
		})

		It("should return nil when the namespace label is missing for a namespaced origin", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid": "uid",
				},
				Annotations: map[string]string{
					"test-origin-name": "name",
				},
			}}

			Expect(namespaced.DataOf(obj)).To(BeNil())
		})
	})

	Describe("StemsFrom", func() {
		It("should return true when the object stems from the source", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{UID: "match-uid"}}
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{"test-origin-uid": "match-uid"},
				Annotations: map[string]string{"test-origin-name": "name"},
			}}

			Expect(clusterScoped.StemsFrom(obj, source)).To(BeTrue())
		})

		It("should return false when UIDs don't match", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{UID: "other-uid"}}
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{"test-origin-uid": "match-uid"},
				Annotations: map[string]string{"test-origin-name": "name"},
			}}

			Expect(clusterScoped.StemsFrom(obj, source)).To(BeFalse())
		})

		It("should return false when the object has no origin data", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{UID: "uid"}}
			obj := &corev1.ConfigMap{}

			Expect(clusterScoped.StemsFrom(obj, source)).To(BeFalse())
		})
	})

	Describe("StemsFromKey", func() {
		It("should return true when name and namespace match the key", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid":       "uid",
					"test-origin-namespace": "src-ns",
				},
				Annotations: map[string]string{
					"test-origin-name": "src-name",
				},
			}}

			Expect(namespaced.StemsFromKey(obj, client.ObjectKey{
				Namespace: "src-ns",
				Name:      "src-name",
			})).To(BeTrue())
		})

		It("should return false when the name doesn't match", func() {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"test-origin-uid":       "uid",
					"test-origin-namespace": "src-ns",
				},
				Annotations: map[string]string{
					"test-origin-name": "src-name",
				},
			}}

			Expect(namespaced.StemsFromKey(obj, client.ObjectKey{
				Namespace: "src-ns",
				Name:      "different-name",
			})).To(BeFalse())
		})

		It("should return false when the object has no origin data", func() {
			obj := &corev1.ConfigMap{}

			Expect(namespaced.StemsFromKey(obj, client.ObjectKey{
				Namespace: "ns",
				Name:      "name",
			})).To(BeFalse())
		})
	})

	Describe("SetOrigin / DataOf round-trip", func() {
		It("should produce data that DataOf can read back correctly", func() {
			source := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Namespace: "round-ns",
				Name:      "round-name",
				UID:       "round-uid",
			}}
			target := &corev1.ConfigMap{}

			namespaced.SetOrigin(source, target)
			data := namespaced.DataOf(target)

			Expect(data).NotTo(BeNil())
			Expect(data.Namespace).To(Equal("round-ns"))
			Expect(data.Name).To(Equal("round-name"))
			Expect(data.UID).To(Equal(types.UID("round-uid")))
		})
	})
})
