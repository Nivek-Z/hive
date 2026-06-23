package tree_test

import (
	"reflect"
	"testing"

	"hive-tui/internal/model"
	"hive-tui/internal/tree"
)

func TestBuildVisibleChannelsKeepsCategoriesAndTextChannelsInPositionOrder(t *testing.T) {
	channels := []model.Channel{
		{ID: 2, Type: "TEXT", Name: "general", Position: 2},
		{ID: 1, Type: "CATEGORY", Name: "main", Position: 1},
		{ID: 4, Type: "DM", Name: "dm", Position: 3},
		{ID: 3, ParentID: ptrInt64(1), Type: "TEXT", Name: "homework", Position: 1},
	}

	got := tree.BuildVisible(channels, map[int64]int{3: 2})
	names := []string{got[0].Name, got[1].Name, got[2].Name}

	if !reflect.DeepEqual(names, []string{"main", "homework", "general"}) {
		t.Fatalf("names = %#v", names)
	}
	if got[0].Depth != 0 || got[1].Depth != 1 || got[1].Unread != 2 {
		t.Fatalf("unexpected visible tree: %#v", got)
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
