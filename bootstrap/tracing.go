package bootstrap

// tracingExcludePaths returns cfg, or the default health path when cfg is empty.
func tracingExcludePaths(cfg []string) []string {
	if len(cfg) == 0 {
		return []string{"/health"}
	}
	return cfg
}
