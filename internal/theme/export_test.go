package theme

import "sync"

// SetUserThemesDirForTest overrides the user themes directory for tests.
func SetUserThemesDirForTest(fn func() string) func() {
	prev := userThemesDir
	userThemesDir = fn
	userThemesOnce = sync.Once{}
	return func() {
		userThemesDir = prev
		userThemesOnce = sync.Once{}
	}
}
