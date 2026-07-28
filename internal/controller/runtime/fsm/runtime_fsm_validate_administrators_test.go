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

package fsm

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("validateAdministrator", func() {
	DescribeTable("should allow valid administrators",
		func(admin string) {
			err := validateAdministrator(admin)
			Expect(err).ShouldNot(HaveOccurred())
		},
		// Email addresses
		Entry("simple email", "user@example.com"),
		Entry("email with dots", "first.last@example.com"),
		Entry("email with plus", "user+tag@example.com"),
		Entry("email with subdomain", "user@mail.example.com"),
		Entry("SAP email", "john.doe@sap.com"),

		// GitHub OIDC tokens
		Entry("github pull_request", "github:repo:org/repo:pull_request"),
		Entry("github push", "github:repo:org/repo:push"),
		Entry("github ref", "github:repo:cx-commerce/cxmc-saastooling-kyma-charts:ref:refs/heads/main"),

		// ArgoCD service accounts
		Entry("argocd application-controller", "argocd:system:serviceaccount:sf-c6f4c9c3-2ead-4feb-810f-5be6ec043171:argocd-application-controller"),
		Entry("argocd server", "argocd:system:serviceaccount:sf-c6f4c9c3-2ead-4feb-810f-5be6ec043171:argocd-server"),

		// MCP service accounts
		Entry("mcp helm-controller", "mcp:system:serviceaccount:flux-system:helm-controller"),
		Entry("mcp kustomize-controller", "mcp:system:serviceaccount:flux-system:kustomize-controller"),
		Entry("mcp provider-kubernetes", "mcp:system:serviceaccount:crossplane-system:provider-kubernetes"),

		// Custom groups (not starting with system:)
		Entry("custom group", "my-custom-group"),
		Entry("custom group with dashes", "team-platform-admins"),
	)

	DescribeTable("should block dangerous system: prefixes",
		func(admin string) {
			err := validateAdministrator(admin)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("system:"))
		},
		Entry("system:anonymous", "system:anonymous"),
		Entry("system:unauthenticated", "system:unauthenticated"),
		Entry("system:authenticated", "system:authenticated"),
		Entry("system:masters", "system:masters"),
		Entry("system:nodes", "system:nodes"),
		Entry("system:node:specific", "system:node:node-1"),
		Entry("system:serviceaccount raw", "system:serviceaccount:kube-system:default"),
		Entry("system:kube-controller-manager", "system:kube-controller-manager"),
		Entry("system:kube-scheduler", "system:kube-scheduler"),
		// Case insensitive
		Entry("SYSTEM:MASTERS uppercase", "SYSTEM:MASTERS"),
		Entry("System:Anonymous mixed case", "System:Anonymous"),
	)

	DescribeTable("should block malformed input",
		func(admin string, expectedErr string) {
			err := validateAdministrator(admin)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(expectedErr))
		},
		Entry("empty string", "", "empty"),
		Entry("whitespace only", "   ", "empty"),
		Entry("tabs only", "\t\t", "empty"),
		Entry("leading space", " user@example.com", "whitespace"),
		Entry("trailing space", "user@example.com ", "whitespace"),
		Entry("leading tab", "\tuser@example.com", "whitespace"),
		Entry("contains newline", "user@example.com\nadmin", "control character"),
		Entry("contains carriage return", "user@example.com\radmin", "control character"),
		Entry("contains tab in middle", "user\t@example.com", "control character"),
		Entry("contains null byte", "user\x00@example.com", "control character"),
	)

	It("should block strings exceeding max length", func() {
		longAdmin := strings.Repeat("a", 254)
		err := validateAdministrator(longAdmin)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("exceeds maximum length"))
	})

	It("should allow strings at max length", func() {
		maxLengthAdmin := strings.Repeat("a", 253)
		err := validateAdministrator(maxLengthAdmin)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

var _ = Describe("validateAdministrators", func() {
	It("should pass for empty list", func() {
		err := validateAdministrators([]string{})
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should pass for nil list", func() {
		err := validateAdministrators(nil)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should pass for valid administrators", func() {
		admins := []string{
			"user@example.com",
			"github:repo:org/repo:pull_request",
			"argocd:system:serviceaccount:ns:sa",
		}
		err := validateAdministrators(admins)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should fail and report index for invalid administrator", func() {
		admins := []string{
			"valid@example.com",
			"system:anonymous",
			"another@example.com",
		}
		err := validateAdministrators(admins)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("administrator[1]"))
		Expect(err.Error()).Should(ContainSubstring("system:anonymous"))
	})
})

func TestValidateAdministrator(t *testing.T) {
	tests := []struct {
		name    string
		admin   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid github oidc", "github:repo:org/repo:pull_request", false},
		{"valid argocd sa", "argocd:system:serviceaccount:ns:name", false},
		{"valid mcp sa", "mcp:system:serviceaccount:ns:name", false},
		{"blocked system:anonymous", "system:anonymous", true},
		{"blocked system:masters", "system:masters", true},
		{"blocked empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAdministrator(tt.admin)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAdministrator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
