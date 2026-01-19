package transcoder

import "testing"

func TestGopArgsFrames(t *testing.T) {
	args, err := gopArgs("60")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 || args[0] != "-g" || args[1] != "60" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestGopArgsDuration(t *testing.T) {
	args, err := gopArgs("2s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "expr:gte(t,n_forced*2.000)"
	if len(args) != 2 || args[0] != "-force_key_frames" || args[1] != want {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestGopArgsInvalid(t *testing.T) {
	cases := []string{"0", "-1", "0s", "nope"}
	for _, c := range cases {
		if args, err := gopArgs(c); err == nil {
			t.Fatalf("expected error for %q, got args %#v", c, args)
		}
	}
}
