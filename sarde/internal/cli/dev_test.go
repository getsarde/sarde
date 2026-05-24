package cli

import "testing"

func TestWatchStdinFlagRegistered(t *testing.T) {
	flag := devCmd.Flags().Lookup("watch-stdin")
	if flag == nil {
		t.Fatal("--watch-stdin flag not registered on serve command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--watch-stdin default = %q, want %q", flag.DefValue, "false")
	}
}
