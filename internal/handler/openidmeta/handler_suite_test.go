// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOpenIDMeta(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenID Metadata Handler Test Suite")
}
