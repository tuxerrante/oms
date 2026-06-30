// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("service CIDR parsing", func() {
	Describe("serviceCIDRFromArgs", func() {
		It("returns the service CIDR when the flag is present", func() {
			args := []string{
				"--advertise-address=10.0.0.1",
				"--service-cluster-ip-range=10.96.0.0/12",
			}
			Expect(serviceCIDRFromArgs(args)).To(Equal("10.96.0.0/12"))
		})

		It("supports split flag/value syntax", func() {
			args := []string{
				"--advertise-address=10.0.0.1",
				"--service-cluster-ip-range",
				"10.99.0.0/12",
			}
			Expect(serviceCIDRFromArgs(args)).To(Equal("10.99.0.0/12"))
		})

		It("returns empty string when the flag is missing", func() {
			Expect(serviceCIDRFromArgs([]string{"--advertise-address=10.0.0.1"})).To(BeEmpty())
		})
	})

	Describe("serviceCIDRFromAPIServerPod", func() {
		It("finds the service CIDR in container command", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Command: []string{
								"kube-apiserver",
								"--service-cluster-ip-range=10.97.0.0/12",
							},
						},
					},
				},
			}
			Expect(serviceCIDRFromAPIServerPod(pod)).To(Equal("10.97.0.0/12"))
		})

		It("finds the service CIDR in container args", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Command: []string{"kube-apiserver"},
							Args:    []string{"--service-cluster-ip-range=10.98.0.0/12"},
						},
					},
				},
			}
			Expect(serviceCIDRFromAPIServerPod(pod)).To(Equal("10.98.0.0/12"))
		})

		It("returns empty string when no service CIDR is configured", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Command: []string{"kube-apiserver", "--advertise-address=10.0.0.1"},
						},
					},
				},
			}
			Expect(serviceCIDRFromAPIServerPod(pod)).To(BeEmpty())
		})
	})

	Describe("serviceCIDRFromProcCmdline", func() {
		It("extracts service CIDR from equals syntax", func() {
			cmdline := "kube-apiserver\x00--service-cluster-ip-range=10.100.0.0/12\x00"
			Expect(serviceCIDRFromProcCmdline(cmdline)).To(Equal("10.100.0.0/12"))
		})

		It("extracts service CIDR from split syntax", func() {
			cmdline := "kube-apiserver\x00--service-cluster-ip-range\x0010.101.0.0/12\x00"
			Expect(serviceCIDRFromProcCmdline(cmdline)).To(Equal("10.101.0.0/12"))
		})
	})
})
