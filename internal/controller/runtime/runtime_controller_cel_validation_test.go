/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtime

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2" //nolint:revive
	. "github.com/onsi/gomega"    //nolint:revive
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("Runtime Administrator CEL Validation", func() {
	ctx := context.Background()

	DescribeTable("should reject Runtime with invalid administrators",
		func(adminName string, expectedErrSubstring string) {
			runtime := CreateRuntimeStub("test-cel-validation")
			runtime.Spec.Security.Administrators = []string{adminName}

			err := k8sClient.Create(ctx, runtime)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsInvalid(err)).To(BeTrue(), "expected Invalid error, got: %v", err)
			Expect(err.Error()).To(ContainSubstring(expectedErrSubstring))
		},
		// system: prefix tests (should be blocked)
		Entry("system:anonymous", "system:anonymous", "system:"),
		Entry("system:authenticated", "system:authenticated", "system:"),
		Entry("system:unauthenticated", "system:unauthenticated", "system:"),
		Entry("system:masters", "system:masters", "system:"),
		Entry("system:nodes", "system:nodes", "system:"),
		Entry("system:serviceaccount:kube-system:default", "system:serviceaccount:kube-system:default", "system:"),
		Entry("SYSTEM:MASTERS (case insensitive)", "SYSTEM:MASTERS", "system:"),
		Entry("System:Anonymous (mixed case)", "System:Anonymous", "system:"),

		// Empty/whitespace tests
		Entry("empty string", "", "empty"),
		Entry("whitespace only", "   ", "empty"),
		Entry("leading space", " user@example.com", "whitespace"),
		Entry("trailing space", "user@example.com ", "whitespace"),
	)

	var validAdminTestCounter int
	DescribeTable("should accept Runtime with valid administrators",
		func(adminName string) {
			validAdminTestCounter++
			runtime := CreateRuntimeStub(fmt.Sprintf("test-cel-valid-%d", validAdminTestCounter))
			runtime.Spec.Security.Administrators = []string{adminName}

			err := k8sClient.Create(ctx, runtime)
			Expect(err).NotTo(HaveOccurred(), "expected no error for admin %q, got: %v", adminName, err)

			// Cleanup
			_ = k8sClient.Delete(ctx, runtime)
		},
		// Valid email addresses
		Entry("simple email", "user@example.com"),
		Entry("email with dots", "first.last@example.com"),

		// Valid service accounts with trusted prefixes
		Entry("argocd service account", "argocd:system:serviceaccount:ns:sa"),
		Entry("mcp service account", "mcp:system:serviceaccount:flux-system:helm-controller"),
		Entry("github OIDC", "github:repo:org/repo:pull_request"),

		// Custom groups (not starting with system:)
		Entry("custom group", "my-custom-group"),
		Entry("group with dashes", "team-platform-admins"),
	)

	It("should reject Runtime when administrators list exceeds max length", func() {
		runtime := CreateRuntimeStub("test-cel-length")
		// Create a string longer than 253 characters
		longAdmin := make([]byte, 254)
		for i := range longAdmin {
			longAdmin[i] = 'a'
		}
		runtime.Spec.Security.Administrators = []string{string(longAdmin)}

		err := k8sClient.Create(ctx, runtime)
		Expect(err).To(HaveOccurred())
		Expect(k8serrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("Too long"))
	})

	It("should accept Runtime when administrator is exactly at max length", func() {
		runtime := CreateRuntimeStub("test-cel-max-length")
		// Create a string exactly 253 characters
		maxLengthAdmin := make([]byte, 253)
		for i := range maxLengthAdmin {
			maxLengthAdmin[i] = 'a'
		}
		runtime.Spec.Security.Administrators = []string{string(maxLengthAdmin)}

		err := k8sClient.Create(ctx, runtime)
		Expect(err).NotTo(HaveOccurred())

		// Cleanup
		_ = k8sClient.Delete(ctx, runtime)
	})

	It("should reject Runtime when one admin in list is invalid", func() {
		runtime := CreateRuntimeStub("test-cel-mixed")
		runtime.Spec.Security.Administrators = []string{
			"valid@example.com",
			"system:masters", // invalid
			"another@example.com",
		}

		err := k8sClient.Create(ctx, runtime)
		Expect(err).To(HaveOccurred())
		Expect(k8serrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("system:"))
	})
})
