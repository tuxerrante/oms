// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/codesphere-cloud/oms/cli/cmd"
)

var _ = Describe("BootstrapLocalCmd", func() {
	Describe("AddBootstrapLocalCmd", func() {
		It("uses the correct pod CIDR flag description", func() {
			parent := &cobra.Command{Use: "beta"}

			cmd.AddBootstrapLocalCmd(parent)

			bootstrapLocalCmd, _, err := parent.Find([]string{"bootstrap-local"})
			Expect(err).NotTo(HaveOccurred())
			Expect(bootstrapLocalCmd).NotTo(BeNil())

			podCIDRFlag := bootstrapLocalCmd.Flags().Lookup("pod-cidr")
			Expect(podCIDRFlag).NotTo(BeNil())
			Expect(podCIDRFlag.Usage).To(Equal("Pod CIDR of the Kubernetes cluster. If not specified, OMS will try to determine it."))
			Expect(bootstrapLocalCmd.Long).To(ContainSubstring("macOS host with a Linux VM-backed Kubernetes cluster"))
		})
	})

	Describe("ConfirmLocalBootstrapWarning", func() {
		It("documents the supported macOS host workflow", func() {
			bootstrapLocalCmd := &cmd.BootstrapLocalCmd{
				Yes: true,
			}

			output := captureStdout(func() {
				Expect(bootstrapLocalCmd.ConfirmLocalBootstrapWarning()).To(Succeed())
			})

			Expect(output).To(ContainSubstring("macOS host with a Linux VM-backed Kubernetes cluster"))
			Expect(output).To(ContainSubstring("Host-native macOS clusters are not supported"))
		})
	})

	Describe("ValidateHostTools", func() {
		It("fails fast when kubectl is missing on non-Linux hosts", func() {
			if runtime.GOOS == "linux" {
				Skip("non-Linux host validation is not active on Linux")
			}

			bootstrapLocalCmd := &cmd.BootstrapLocalCmd{}
			restorePath := setPathForTest(GinkgoT().TempDir())
			defer restorePath()

			err := bootstrapLocalCmd.ValidateHostTools()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("kubectl"))
		})

		It("fails fast when node is missing on non-Linux hosts", func() {
			if runtime.GOOS == "linux" {
				Skip("non-Linux host validation is not active on Linux")
			}

			bootstrapLocalCmd := &cmd.BootstrapLocalCmd{}
			tempDir := GinkgoT().TempDir()
			Expect(writeExecutable(tempDir, "kubectl")).To(Succeed())
			restorePath := setPathForTest(tempDir)
			defer restorePath()

			err := bootstrapLocalCmd.ValidateHostTools()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("node"))
		})

		It("accepts non-Linux hosts when kubectl and node are available", func() {
			if runtime.GOOS == "linux" {
				Skip("non-Linux host validation is not active on Linux")
			}

			bootstrapLocalCmd := &cmd.BootstrapLocalCmd{}
			tempDir := GinkgoT().TempDir()
			Expect(writeExecutable(tempDir, "kubectl")).To(Succeed())
			Expect(writeExecutable(tempDir, "node")).To(Succeed())
			restorePath := setPathForTest(tempDir)
			defer restorePath()

			Expect(bootstrapLocalCmd.ValidateHostTools()).To(Succeed())
		})
	})
})

func captureStdout(fn func()) string {
	originalStdout := os.Stdout

	reader, writer, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	Expect(writer.Close()).To(Succeed())

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	Expect(err).NotTo(HaveOccurred())
	Expect(reader.Close()).To(Succeed())

	return buf.String()
}

func setPathForTest(path string) func() {
	originalPath, exists := os.LookupEnv("PATH")
	Expect(os.Setenv("PATH", path)).To(Succeed())

	return func() {
		if exists {
			Expect(os.Setenv("PATH", originalPath)).To(Succeed())
			return
		}

		Expect(os.Unsetenv("PATH")).To(Succeed())
	}
}

func writeExecutable(dir, name string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755)
}
