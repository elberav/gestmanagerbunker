//go:build windows && !debug

package backend

func init() {
	hideFromDebugger()
}
