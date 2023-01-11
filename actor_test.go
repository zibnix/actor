package actor

import "testing"

type method struct {
	field string
}

func (m *method) meth(v string) {
	m.field = v
}

func TestMethodVars(t *testing.T) {
	m := &method{}
	var f func(string)

	f = m.meth

	f("v1")

	if m.field != "v1" {
		t.Fatal("method variable failed to update field")
	}

	f("v2")

	if m.field != "v2" {
		t.Fatal("method variable failed to update field")
	}
}
