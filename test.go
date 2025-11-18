package main

import (
	"fmt"
	"regexp"
)
func main() {
	roleArn := "arn:aws:iam::123456789012:role/MyRole"
	re := regexp.MustCompile(`^arn:aws:iam::.+:role\/(.+)$`)
	matches := re.FindStringSubmatch(roleArn)
	fmt.Println(matches)
	// if len(matches) == 2 {
	// 	return matches[1]
	// }
	// return ""	
}