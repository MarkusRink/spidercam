package output

import "testing"

func TestNullSinkModuleArgs(t *testing.T) {
	got := NullSinkModuleArgs("spidercam_sink")
	want := "sink_name=spidercam_sink sink_properties=device.description=Spidercam_Virtual_Mic"
	if got != want {
		t.Fatalf("NullSinkModuleArgs() = %q, want %q", got, want)
	}
}

func TestNullSinkModuleArgsCustomName(t *testing.T) {
	got := NullSinkModuleArgs("custom_sink")
	if got != "sink_name=custom_sink sink_properties=device.description=Spidercam_Virtual_Mic" {
		t.Fatalf("NullSinkModuleArgs() = %q", got)
	}
}
