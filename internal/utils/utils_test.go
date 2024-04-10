// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"errors"

	"github.com/gardener/gardener-discovery-server/internal/utils"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("#SplitProjectNameAndShootUID", func() {
		It("should correctly split project name and shoot uid", func() {
			projName := "test"
			uid := uuid.New().String()
			name, id, err := utils.SplitProjectNameAndShootUID(projName + "--" + uid)
			Expect(err).To(Not(HaveOccurred()))
			Expect(name).To(Equal(projName))
			Expect(id).To(Equal(uid))
		})

		DescribeTable(
			"should produce an error",
			func(input string) {
				name, id, err := utils.SplitProjectNameAndShootUID(input)
				Expect(errors.Is(err, utils.ErrProjShootUIDInvalidFormat)).To(BeTrue())
				Expect(name).To(BeEmpty())
				Expect(id).To(BeEmpty())
			},
			Entry("empty string", ""),
			Entry("just a delimiter", "--"),
			Entry("delimiter with prefix", "a--"),
			Entry("delimiter with suffix", "--a"),
			Entry("no delimiter", "foo"),
			Entry("too many arguments to split", "a--b--c"),
		)
	})
})
