//go:build linux && !debug

package backend

func init() {
	earlyAntiDebug()
}
