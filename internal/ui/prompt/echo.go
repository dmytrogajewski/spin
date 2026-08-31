package prompt

// FormatUserEcho renders a submitted prompt so it stays distinct from agent work.
func FormatUserEcho(line string) string {
	return ColorUserEcho + DefaultPrefix + line + ansiReset
}
