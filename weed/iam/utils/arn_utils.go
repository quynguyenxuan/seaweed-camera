package utils

import (
	"regexp"
	"strings"
)

// ExtractRoleNameFromPrincipal extracts role name from principal ARN
// Handles both STS assumed role and IAM role formats
func ExtractRoleNameFromPrincipalPrefix(principal string) string {
	// Handle STS assumed role format: arn:seaweed:sts::assumed-role/RoleName/SessionName
	stsPrefix := "arn:aws:sts::assumed-role/"
	if strings.HasPrefix(principal, stsPrefix) {
		remainder := principal[len(stsPrefix):]
		// Split on first '/' to get role name
		if slashIndex := strings.Index(remainder, "/"); slashIndex != -1 {
			return remainder[:slashIndex]
		}
		// If no slash found, return the remainder (edge case)
		return remainder
	}

	// Handle IAM role format: arn:seaweed:iam::role/RoleName
	iamPrefix := "arn:aws:iam::role/"
	if strings.HasPrefix(principal, iamPrefix) {
		return principal[len(iamPrefix):]
	}

	// Return empty string to signal invalid ARN format
	// This allows callers to handle the error explicitly instead of masking it
	return ""
}
// ExtractRoleNameFromPrincipal extracts role name from principal ARN
// Handles both STS assumed role and IAM role formats
func ExtractRoleNameFromPrincipal(principal string) string {
	// Handle STS assumed role format: arn:aws:sts::123456789012:assumed-role/RoleName/SessionName
	reSTS := regexp.MustCompile(`^arn:aws:sts::?.*:assumed-role\/([^/]+)\/([^/]+)$`)
	if matches := reSTS.FindStringSubmatch(principal); len(matches) >= 2 {
		return matches[1]
	}

	// Handle IAM role format: arn:aws:iam::123456789012:role/RoleName
	reIAM := regexp.MustCompile(`^arn:aws:iam::?.*:role\/([^/]+)$`)
	if matches := reIAM.FindStringSubmatch(principal); len(matches) >= 2 {
		return matches[1]
	}
	// Return empty string to signal invalid ARN format
	return ""
}

// ExtractRoleNameFromArn extracts role name from an IAM role ARN
// Handles format: arn:aws:iam::123456789012:role/RoleName
func ExtractRoleNameFromArn(roleArn string) string {
	re := regexp.MustCompile(`^arn:aws:iam::?.*:role\/([^/]+)$`)
	matches := re.FindStringSubmatch(roleArn)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func ExtractUidAndRoleNameFromArn(roleArn string) (string, string) {
	re := regexp.MustCompile(`^arn:aws:iam::?(.*):role\/([^/]+)$`)
	matches := re.FindStringSubmatch(roleArn)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return "", ""
}

// ExtractRoleNameFromArn extracts role name from an IAM role ARN
// Specifically handles: arn:seaweed:iam::role/RoleName
func ExtractRoleNameFromArnPrefix(roleArn string) string {
	prefix := "arn:aws:iam::role/"
	if strings.HasPrefix(roleArn, prefix) && len(roleArn) > len(prefix) {
		return roleArn[len(prefix):]
	}
	return ""
}
