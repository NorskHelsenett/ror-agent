package services

import (
	"reflect"
	"testing"
)

func TestNewAccessGroupsFromData_NilData(t *testing.T) {
	got := NewAccessGroupsFromData(nil)
	want := accessGroups{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestNewAccessGroupsFromData_EmptyData(t *testing.T) {
	data := map[string]string{}
	got := NewAccessGroupsFromData(data)
	want := accessGroups{
		accessGroups:          []string{""},
		readOnlyAccessGroups:  []string{""},
		grafanaAdminGroups:    []string{""},
		grafanaReadOnlyGroups: []string{""},
		argocdAdminGroups:     []string{""},
		argocdReadOnlyGroups:  []string{""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestNewAccessGroupsFromData_SingleValues(t *testing.T) {
	data := map[string]string{
		"accessGroups":          "foo",
		"readOnlyAccessGroups":  "bar",
		"grafanaAdminGroups":    "baz",
		"grafanaReadOnlyGroups": "qux",
		"argocdAdminGroups":     "quux",
		"argocdReadOnlyGroups":  "corge",
	}
	got := NewAccessGroupsFromData(data)
	want := accessGroups{
		accessGroups:          []string{"foo"},
		readOnlyAccessGroups:  []string{"bar"},
		grafanaAdminGroups:    []string{"baz"},
		grafanaReadOnlyGroups: []string{"qux"},
		argocdAdminGroups:     []string{"quux"},
		argocdReadOnlyGroups:  []string{"corge"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestNewAccessGroupsFromData_MultipleValues(t *testing.T) {
	data := map[string]string{
		"accessGroups":          "foo;bar",
		"readOnlyAccessGroups":  "baz;qux",
		"grafanaAdminGroups":    "quux;corge",
		"grafanaReadOnlyGroups": "grault;garply",
		"argocdAdminGroups":     "waldo;fred",
		"argocdReadOnlyGroups":  "plugh;xyzzy",
	}
	got := NewAccessGroupsFromData(data)
	want := accessGroups{
		accessGroups:          []string{"foo", "bar"},
		readOnlyAccessGroups:  []string{"baz", "qux"},
		grafanaAdminGroups:    []string{"quux", "corge"},
		grafanaReadOnlyGroups: []string{"grault", "garply"},
		argocdAdminGroups:     []string{"waldo", "fred"},
		argocdReadOnlyGroups:  []string{"plugh", "xyzzy"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestNewAccessGroupsFromData_MissingKeys(t *testing.T) {
	data := map[string]string{
		"accessGroups": "foo",
	}
	got := NewAccessGroupsFromData(data)
	want := accessGroups{
		accessGroups:          []string{"foo"},
		readOnlyAccessGroups:  []string{""},
		grafanaAdminGroups:    []string{""},
		grafanaReadOnlyGroups: []string{""},
		argocdAdminGroups:     []string{""},
		argocdReadOnlyGroups:  []string{""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}
func TestAccessGroups_StringArray_AllEmpty(t *testing.T) {
	a := accessGroups{
		accessGroups:          []string{""},
		readOnlyAccessGroups:  []string{""},
		grafanaAdminGroups:    []string{""},
		grafanaReadOnlyGroups: []string{""},
		argocdAdminGroups:     []string{""},
		argocdReadOnlyGroups:  []string{""},
	}
	got := a.StringArray()
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestAccessGroups_StringArray_SingleValues(t *testing.T) {
	a := accessGroups{
		accessGroups:          []string{"foo"},
		readOnlyAccessGroups:  []string{"bar"},
		grafanaAdminGroups:    []string{"baz"},
		grafanaReadOnlyGroups: []string{"qux"},
		argocdAdminGroups:     []string{"quux"},
		argocdReadOnlyGroups:  []string{"corge"},
	}
	got := a.StringArray()
	want := []string{
		"Cluster Operator - foo",
		"Cluster ReadOnly - bar",
		"Grafana Operator - baz",
		"Grafana ReadOnly - qux",
		"ArgoCD Operator - quux",
		"ArgoCD ReadOnly - corge",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestAccessGroups_StringArray_MultipleValues(t *testing.T) {
	a := accessGroups{
		accessGroups:          []string{"foo", "bar"},
		readOnlyAccessGroups:  []string{"baz", "qux"},
		grafanaAdminGroups:    []string{"quux", "corge"},
		grafanaReadOnlyGroups: []string{"grault", "garply"},
		argocdAdminGroups:     []string{"waldo", "fred"},
		argocdReadOnlyGroups:  []string{"plugh", "xyzzy"},
	}
	got := a.StringArray()
	want := []string{
		"Cluster Operator - foo",
		"Cluster Operator - bar",
		"Cluster ReadOnly - baz",
		"Cluster ReadOnly - qux",
		"Grafana Operator - quux",
		"Grafana Operator - corge",
		"Grafana ReadOnly - grault",
		"Grafana ReadOnly - garply",
		"ArgoCD Operator - waldo",
		"ArgoCD Operator - fred",
		"ArgoCD ReadOnly - plugh",
		"ArgoCD ReadOnly - xyzzy",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestAccessGroups_StringArray_SomeEmptyValues(t *testing.T) {
	a := accessGroups{
		accessGroups:          []string{"foo", ""},
		readOnlyAccessGroups:  []string{"", "bar"},
		grafanaAdminGroups:    []string{""},
		grafanaReadOnlyGroups: []string{"qux", ""},
		argocdAdminGroups:     []string{""},
		argocdReadOnlyGroups:  []string{"corge"},
	}
	got := a.StringArray()
	want := []string{
		"Cluster Operator - foo",
		"Cluster ReadOnly - bar",
		"Grafana ReadOnly - qux",
		"ArgoCD ReadOnly - corge",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}
