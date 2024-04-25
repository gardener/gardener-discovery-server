// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"strings"
)

// ErrProjShootUIDInvalidFormat is an error that is returned if
// an issuer metadata shoot secret name is not in the correct format.
var ErrProjShootUIDInvalidFormat = errors.New("input not in the correct format: projectName--shootUID")

// SplitProjectNameAndShootUID splits the key by '--' in two parts.
func SplitProjectNameAndShootUID(key string) (string, string, error) {
	split := strings.Split(key, "--")
	if len(split) != 2 || strings.TrimSpace(split[0]) == "" || strings.TrimSpace(split[1]) == "" {
		return "", "", ErrProjShootUIDInvalidFormat
	}
	return split[0], split[1], nil
}
