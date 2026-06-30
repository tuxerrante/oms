// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package local

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("installer host behavior", func() {
	Describe("shouldReadServiceCIDRFromProc", func() {
		It("uses proc fallback on Linux hosts only", func() {
			Expect(shouldReadServiceCIDRFromProc("linux")).To(BeTrue())
			Expect(shouldReadServiceCIDRFromProc("darwin")).To(BeFalse())
		})
	})

	Describe("useLocalPortForwardForInstaller", func() {
		It("prefers a localhost port-forward on non-Linux hosts", func() {
			Expect(useLocalPortForwardForInstaller("darwin")).To(BeTrue())
			Expect(useLocalPortForwardForInstaller("linux")).To(BeFalse())
		})
	})

	Describe("buildPostgresPortForwardArgs", func() {
		It("targets the postgres service on localhost", func() {
			Expect(buildPostgresPortForwardArgs(15432, 5432)).To(Equal([]string{
				"-n", codesphereNamespace,
				"port-forward",
				"svc/masterdata-rw",
				"15432:5432",
				"--address", "127.0.0.1",
			}))
		})
	})
})
