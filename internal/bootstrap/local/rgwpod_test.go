// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/codesphere-cloud/oms/internal/bootstrap"
)

var rgwServiceName = "rook-ceph-rgw-" + rgwObjectStoreName

func newRGWService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: rgwServiceName, Namespace: rookNamespace},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "rook-ceph-rgw"}},
	}
}

func newRGWPod(phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-rgw-pod",
			Namespace: rookNamespace,
			Labels:    map[string]string{"app": "rook-ceph-rgw"},
		},
		Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "rgw"}}},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func newBootstrapper(kubeClient client.Client) *LocalBootstrapper {
	return &LocalBootstrapper{
		ctx:        context.Background(),
		stlog:      bootstrap.NewStepLogger(true),
		kubeClient: kubeClient,
	}
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	return scheme
}

var _ = Describe("RGW pod lookup", func() {
	Describe("getRGWPod", func() {
		It("returns a running RGW pod", func() {
			kubeClient := fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithObjects(newRGWService(), newRGWPod(corev1.PodRunning)).
				Build()

			pod, err := newBootstrapper(kubeClient).getRGWPod(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal("rook-ceph-rgw-pod"))
		})

		DescribeTable("classifies transient states as retryable",
			func(objects ...client.Object) {
				kubeClient := fake.NewClientBuilder().
					WithScheme(newScheme()).
					WithObjects(objects...).
					Build()

				_, err := newBootstrapper(kubeClient).getRGWPod(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(isRetryableWaitError(err)).To(BeTrue())
			},
			Entry("missing service"),
			Entry("service without selector", client.Object(&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: rgwServiceName, Namespace: rookNamespace},
			})),
			Entry("no pods", client.Object(newRGWService())),
			Entry("pod not running", client.Object(newRGWService()), client.Object(newRGWPod(corev1.PodPending))),
		)

		It("returns terminal errors when getting the service fails", func() {
			forbidden := apierrors.NewForbidden(
				schema.GroupResource{Resource: "services"}, rgwServiceName, errors.New("nope"))
			kubeClient := fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return forbidden
					},
				}).
				Build()

			_, err := newBootstrapper(kubeClient).getRGWPod(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(isRetryableWaitError(err)).To(BeFalse())
			Expect(apierrors.IsForbidden(err)).To(BeTrue())
		})

		It("returns terminal errors when listing pods fails", func() {
			unauthorized := apierrors.NewUnauthorized("no token")
			kubeClient := fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithObjects(newRGWService()).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
						return unauthorized
					},
				}).
				Build()

			_, err := newBootstrapper(kubeClient).getRGWPod(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(isRetryableWaitError(err)).To(BeFalse())
			Expect(apierrors.IsUnauthorized(err)).To(BeTrue())
		})
	})

	Describe("waitForRGWPod", func() {
		It("aborts immediately on terminal Kubernetes errors", func() {
			forbidden := apierrors.NewForbidden(
				schema.GroupResource{Resource: "services"}, rgwServiceName, errors.New("nope"))
			kubeClient := fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return forbidden
					},
				}).
				Build()

			start := time.Now()
			_, err := newBootstrapper(kubeClient).waitForRGWPod()
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsForbidden(err)).To(BeTrue())
			Expect(time.Since(start)).To(BeNumerically("<", cephReadyPollInterval))
		})

		It("returns the pod once it is running", func() {
			kubeClient := fake.NewClientBuilder().
				WithScheme(newScheme()).
				WithObjects(newRGWService(), newRGWPod(corev1.PodRunning)).
				Build()

			pod, err := newBootstrapper(kubeClient).waitForRGWPod()
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal("rook-ceph-rgw-pod"))
		})
	})

	Describe("retryWithBackoff", func() {
		It("preserves the last transient cause in timeout errors", func() {
			bootstrapper := newBootstrapper(fake.NewClientBuilder().WithScheme(newScheme()).Build())

			err := bootstrapper.retryWithBackoff(time.Millisecond, "timed out waiting",
				func(context.Context) error {
					return &retryableWaitError{err: errors.New("RGW service not found yet")}
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timed out waiting"))
			Expect(err.Error()).To(ContainSubstring("RGW service not found yet"))
		})
	})
})
