package transcriptscan

import "regexp"

var urlPattern = regexp.MustCompile(`https?://\S+`)

// absolutePathPattern matches a whitespace-delimited token that looks
// like a Unix absolute path: starts with /, has at least one more /
// after the first path segment. Deliberately doesn't try to distinguish
// this from a URL's path component — containsURL is checked separately
// and both facts can be true of the same prompt.
var absolutePathPattern = regexp.MustCompile(`(?:^|\s)(/[^\s/]+/[^\s]*)`)

func containsURL(text string) bool {
	return urlPattern.MatchString(text)
}

func containsAbsolutePath(text string) bool {
	return absolutePathPattern.MatchString(text)
}
