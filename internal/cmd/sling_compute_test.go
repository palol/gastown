package cmd

import "testing"

func TestNormalizeComputeTarget(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: computeTargetAuto},
		{in: "auto", want: computeTargetAuto},
		{in: "local", want: computeTargetLocal},
		{in: "bigfoot", want: computeTargetBigfoot},
		{in: "BIGFOOT", want: computeTargetBigfoot},
		{in: "gpu", wantErr: true},
	}

	for _, tt := range tests {
		got, err := normalizeComputeTarget(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("normalizeComputeTarget(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("normalizeComputeTarget(%q)=%q want=%q", tt.in, got, tt.want)
		}
	}
}

func TestResolveComputeTarget(t *testing.T) {
	if got := resolveComputeTarget(computeTargetAuto, []string{"priority:P1", "compute:bigfoot"}); got != computeTargetBigfoot {
		t.Fatalf("auto+label got=%q want=%q", got, computeTargetBigfoot)
	}
	if got := resolveComputeTarget(computeTargetAuto, []string{"priority:P1"}); got != computeTargetLocal {
		t.Fatalf("auto without label got=%q want=%q", got, computeTargetLocal)
	}
	if got := resolveComputeTarget(computeTargetLocal, []string{"compute:bigfoot"}); got != computeTargetLocal {
		t.Fatalf("local override got=%q want=%q", got, computeTargetLocal)
	}
	if got := resolveComputeTarget(computeTargetBigfoot, nil); got != computeTargetBigfoot {
		t.Fatalf("bigfoot override got=%q want=%q", got, computeTargetBigfoot)
	}
}

func TestTargetRigForCompute(t *testing.T) {
	if got := targetRigForCompute("ld_agent_lab", computeTargetBigfoot); got != computeTargetBigfoot {
		t.Fatalf("targetRigForCompute(bigfoot)=%q want=%q", got, computeTargetBigfoot)
	}
	if got := targetRigForCompute("ld_agent_lab", computeTargetLocal); got != "ld_agent_lab" {
		t.Fatalf("targetRigForCompute(local)=%q want=%q", got, "ld_agent_lab")
	}
}
