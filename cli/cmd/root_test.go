// Copyright (c) Codesphere Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/codesphere-cloud/oms/cli/cmd"
)

var _ = Describe("AddCmd", func() {
	It("inherits the parent Args validator when the child does not define one", func() {
		parent := &cobra.Command{
			Use:           "root",
			Args:          cobra.NoArgs,
			SilenceErrors: true,
			SilenceUsage:  true,
		}
		child := &cobra.Command{
			Use:  "child",
			RunE: func(_ *cobra.Command, _ []string) error { return nil },
		}
		cmd.AddCmd(parent, child)

		parent.SetArgs([]string{"child", "extra"})
		err := parent.Execute()

		Expect(err).To(HaveOccurred())
		Expect(parent.Commands()).To(ContainElement(child))
	})

	It("keeps a child-specific Args validator when one is explicitly set", func() {
		parent := &cobra.Command{
			Use:           "root",
			Args:          cobra.NoArgs,
			SilenceErrors: true,
			SilenceUsage:  true,
		}

		capturedArgs := []string{}
		child := &cobra.Command{
			Use:  "child",
			Args: cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				capturedArgs = args
				return nil
			},
		}
		cmd.AddCmd(parent, child)

		parent.SetArgs([]string{"child", "value"})
		err := parent.Execute()

		Expect(err).NotTo(HaveOccurred())
		Expect(capturedArgs).To(Equal([]string{"value"}))
		Expect(parent.Commands()).To(ContainElement(child))
	})
})

var _ = Describe("GetRootCmd", func() {
	It("does not log the upgraded API key value", func() {
		oldKey := "1234567890123456789012345"
		Expect(len(oldKey)).To(Equal(25))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodGet))
			Expect(r.URL.Path).To(Equal("/api/key"))
			Expect(r.Header.Get("X-API-Key")).To(Equal(oldKey))
			_, err := w.Write([]byte(`{"keyId":"pref-"}`))
			Expect(err).NotTo(HaveOccurred())
		}))
		DeferCleanup(server.Close)

		originalKey, hadKey := os.LookupEnv("OMS_PORTAL_API_KEY")
		originalAPI, hadAPI := os.LookupEnv("OMS_PORTAL_API")
		Expect(os.Setenv("OMS_PORTAL_API_KEY", oldKey)).To(Succeed())
		Expect(os.Setenv("OMS_PORTAL_API", server.URL+"/api")).To(Succeed())
		DeferCleanup(func() {
			if hadKey {
				Expect(os.Setenv("OMS_PORTAL_API_KEY", originalKey)).To(Succeed())
			} else {
				Expect(os.Unsetenv("OMS_PORTAL_API_KEY")).To(Succeed())
			}
			if hadAPI {
				Expect(os.Setenv("OMS_PORTAL_API", originalAPI)).To(Succeed())
			} else {
				Expect(os.Unsetenv("OMS_PORTAL_API")).To(Succeed())
			}
		})

		var logs bytes.Buffer
		originalWriter := log.Writer()
		originalFlags := log.Flags()
		log.SetOutput(&logs)
		log.SetFlags(0)
		DeferCleanup(func() {
			log.SetOutput(originalWriter)
			log.SetFlags(originalFlags)
		})

		rootCmd := cmd.GetRootCmd()
		rootCmd.SetArgs([]string{"version"})
		Expect(rootCmd.Execute()).To(Succeed())

		output := logs.String()
		Expect(output).To(ContainSubstring("Please update your OMS_PORTAL_API_KEY environment variable with the upgraded key value."))
		Expect(output).To(ContainSubstring("export OMS_PORTAL_API_KEY='<upgraded-api-key>'"))
		Expect(output).NotTo(ContainSubstring(oldKey))
		Expect(output).NotTo(ContainSubstring("pref-"))
		Expect(output).NotTo(ContainSubstring("pref-" + oldKey))
	})
})
