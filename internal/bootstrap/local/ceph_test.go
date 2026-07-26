// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rookcephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Ceph", func() {
	Describe("cephHostsFromObjectStore", func() {
		It("parses hosts from a valid object store", func() {
			store := &rookcephv1.CephObjectStore{}
			store.Name = "s3-ms-provider"
			store.Status = &rookcephv1.ObjectStoreStatus{
				Endpoints: rookcephv1.ObjectEndpoints{
					Insecure: []string{
						"http://rook-ceph-rgw-s3-ms-provider.rook-ceph.svc:80",
					},
				},
			}

			hosts, err := cephHostsFromObjectStore(store)
			Expect(err).NotTo(HaveOccurred())
			Expect(hosts).To(HaveLen(1))

			expectedHost := "rook-ceph-rgw-s3-ms-provider.rook-ceph.svc"
			Expect(hosts[0].Hostname).To(Equal(expectedHost))
			Expect(hosts[0].IPAddress).To(Equal(expectedHost))
			Expect(hosts[0].IsMaster).To(BeTrue())
		})

		It("rejects invalid endpoint entries", func() {
			store := &rookcephv1.CephObjectStore{}
			store.Name = "s3-ms-provider"
			store.Status = &rookcephv1.ObjectStoreStatus{
				Endpoints: rookcephv1.ObjectEndpoints{
					Insecure: []string{"not-a-url"},
				},
			}

			_, err := cephHostsFromObjectStore(store)
			Expect(err).To(HaveOccurred())
		})

		It("deduplicates endpoints with the same hostname", func() {
			store := &rookcephv1.CephObjectStore{}
			store.Name = "s3-ms-provider"
			store.Status = &rookcephv1.ObjectStoreStatus{
				Endpoints: rookcephv1.ObjectEndpoints{
					Insecure: []string{
						"http://rgw.rook-ceph.svc:80",
						"http://rgw.rook-ceph.svc:8080",
					},
				},
			}

			hosts, err := cephHostsFromObjectStore(store)
			Expect(err).NotTo(HaveOccurred())
			Expect(hosts).To(HaveLen(1))
			Expect(hosts[0].Hostname).To(Equal("rgw.rook-ceph.svc"))
		})
	})

	Describe("parseObjectStoreEndpointHost", func() {
		It("extracts hostname from a valid endpoint", func() {
			host, err := parseObjectStoreEndpointHost("http://rook-ceph-rgw.rook-ceph.svc:80")
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("rook-ceph-rgw.rook-ceph.svc"))
		})

		It("trims whitespace before parsing", func() {
			host, err := parseObjectStoreEndpointHost("  http://rgw.svc:80  \n")
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("rgw.svc"))
		})

		It("rejects a schemeless string", func() {
			_, err := parseObjectStoreEndpointHost("not-a-url")
			Expect(err).To(HaveOccurred())
		})

		It("rejects a bare path with no hostname", func() {
			_, err := parseObjectStoreEndpointHost("/relative/path")
			Expect(err).To(HaveOccurred())
		})

		It("rejects an empty string", func() {
			_, err := parseObjectStoreEndpointHost("")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("rgwUserCredentialsFromSecret", func() {
		It("parses credentials from a valid secret", func() {
			secret := &corev1.Secret{}
			secret.Name = "rgw-admin-user"
			secret.Data = map[string][]byte{
				"AccessKey": []byte("access-key"),
				"SecretKey": []byte("secret-key"),
			}

			credentials, err := rgwUserCredentialsFromSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(credentials.AccessKey).To(Equal("access-key"))
			Expect(credentials.SecretKey).To(Equal("secret-key"))
		})

		It("rejects secrets with missing keys", func() {
			secret := &corev1.Secret{}
			secret.Name = "rgw-admin-user"
			secret.Data = map[string][]byte{
				"username": []byte("foo"),
			}

			_, err := rgwUserCredentialsFromSecret(secret)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("rgwUserCredentialsFromAdminJSON", func() {
		It("parses credentials from valid admin JSON", func() {
			credentials, err := rgwUserCredentialsFromAdminJSON(`{"keys":[{"access_key":"access-key","secret_key":"secret-key"}]}`)
			Expect(err).NotTo(HaveOccurred())
			Expect(credentials.AccessKey).To(Equal("access-key"))
			Expect(credentials.SecretKey).To(Equal("secret-key"))
		})

		It("rejects admin JSON with missing keys", func() {
			_, err := rgwUserCredentialsFromAdminJSON(`{"keys":[]}`)
			Expect(err).To(HaveOccurred())
		})
	})
})
