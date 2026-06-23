package tree

import (
	"sort"

	"hive-tui/internal/model"
)

type VisibleChannel struct {
	ID       int64
	Type     string
	Name     string
	Depth    int
	Unread   int
	Position int
}

func BuildVisible(channels []model.Channel, unreads map[int64]int) []VisibleChannel {
	byParent := map[int64][]model.Channel{}
	for _, channel := range channels {
		if channel.Type != "CATEGORY" && channel.Type != "TEXT" {
			continue
		}
		parent := int64(0)
		if channel.ParentID != nil {
			parent = *channel.ParentID
		}
		byParent[parent] = append(byParent[parent], channel)
	}

	for parent := range byParent {
		sort.SliceStable(byParent[parent], func(i, j int) bool {
			if byParent[parent][i].Position == byParent[parent][j].Position {
				return byParent[parent][i].ID < byParent[parent][j].ID
			}
			return byParent[parent][i].Position < byParent[parent][j].Position
		})
	}

	var visible []VisibleChannel
	var appendLevel func(parent int64, depth int)
	appendLevel = func(parent int64, depth int) {
		for _, channel := range byParent[parent] {
			visible = append(visible, VisibleChannel{
				ID:       channel.ID,
				Type:     channel.Type,
				Name:     channel.Name,
				Depth:    depth,
				Unread:   unreads[channel.ID],
				Position: channel.Position,
			})
			if channel.Type == "CATEGORY" {
				appendLevel(channel.ID, depth+1)
			}
		}
	}
	appendLevel(0, 0)
	return visible
}
