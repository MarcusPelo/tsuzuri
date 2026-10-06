package theme

import "sync"

// SetUserThemesDirForTest overrides the user themes directory for tests.
func SetUserThemesDirForTest(fn func() string) func() {
	prev := userThemesPath
	userThemesPath = fn
	userThemesOnce = sync.Once{}
	userThemes = nil
	return func() {
		userThemesPath = prev
		userThemesOnce = sync.Once{}
		userThemes = nil
	}
}
