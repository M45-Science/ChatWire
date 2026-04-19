package modupdate

import "testing"

func TestDependencySatisfiedUnversionedDepPresent(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep"}, "1.0.0", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected unversioned installed dependency to satisfy requirement")
	}
}

func TestDependencySatisfiedUnversionedDepMissing(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep"}, "", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if good {
		t.Fatal("expected missing unversioned dependency to fail")
	}
}

func TestDependencySatisfiedVersionedDepSatisfiedByInstalledVersion(t *testing.T) {
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.3.0", nil)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected installed version to satisfy dependency")
	}
}

func TestDependencySatisfiedVersionedDepSatisfiedByPlannedUpdate(t *testing.T) {
	planned := []downloadData{{Name: "dep", Version: "1.4.0"}}
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.0.0", planned)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if !good {
		t.Fatal("expected planned download version to satisfy dependency")
	}
}

func TestDependencySatisfiedVersionedDepUnsatisfiedAfterRecursion(t *testing.T) {
	planned := []downloadData{{Name: "dep", Version: "1.1.0"}}
	good, err := dependencySatisfied(depRequires{name: "dep", equality: EO_GREATEREQ, version: "1.2.0"}, "1.0.0", planned)
	if err != nil {
		t.Fatalf("dependencySatisfied returned error: %v", err)
	}
	if good {
		t.Fatal("expected dependency to remain unsatisfied after recursion")
	}
}
