//go:build !tools

package scripts

// zz_dummy.go - keep `nofx/scripts` buildable during normal builds.
// The actual script files are marked with `//go:build tools` and are
// excluded from normal builds and tests. This file intentionally does
// nothing.
