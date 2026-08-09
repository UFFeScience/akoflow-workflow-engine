package mapper

import "testing"

type sourceChild struct{ Value string }
type source struct {
	Name     string
	Ignored  int
	Child    sourceChild
	Values   []string
	Children []sourceChild
}
type targetChild struct{ Value string }
type target struct {
	Name     string
	Ignored  string
	Child    targetChild
	Values   []string
	Children []targetChild
}

func TestMapStructsCopiesCompatibleNestedFieldsAndSlices(t *testing.T) {
	src := source{Name: "name", Ignored: 9, Child: sourceChild{Value: "child"}, Values: []string{"a", "b"}, Children: []sourceChild{{Value: "one"}}}
	var dst target
	MapStructs(src, &dst)
	if dst.Name != "name" || dst.Ignored != "" || dst.Child.Value != "child" || len(dst.Values) != 2 || dst.Values[1] != "b" || len(dst.Children) != 1 || dst.Children[0].Value != "one" {
		t.Fatalf("unexpected mapping: %+v", dst)
	}
}

func TestMapStructsSkipsMissingAndIncompatibleSlice(t *testing.T) {
	var dst struct{ Values string }
	MapStructs(struct{ Values []int }{Values: []int{1}}, &dst)
	if dst.Values != "" {
		t.Fatal("incompatible field must remain unchanged")
	}
}
