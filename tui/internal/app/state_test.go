package app_test

import (
	"testing"

	"hive-tui/internal/app"
	"hive-tui/internal/model"
)

func TestStateSelectChannelClearsMessagesAndUnread(t *testing.T) {
	st := app.State{
		CurrentChannelID: 1,
		Messages:         []model.Message{{ID: 10, ChannelID: 1, Content: "old"}},
		Unreads:          map[int64]int{2: 3},
	}

	st.SelectChannel(2)

	if st.CurrentChannelID != 2 || len(st.Messages) != 0 || st.Unreads[2] != 0 {
		t.Fatalf("state = %#v", st)
	}
}

func TestStateAppendsIncomingMessageForCurrentChannel(t *testing.T) {
	st := app.State{CurrentChannelID: 2, Unreads: map[int64]int{}}

	st.ApplyIncomingMessage(model.Message{ID: 10, ChannelID: 2, Content: "hello"})

	if len(st.Messages) != 1 || st.Messages[0].Content != "hello" {
		t.Fatalf("messages = %#v", st.Messages)
	}
}

func TestStateIncrementsUnreadForOtherChannel(t *testing.T) {
	st := app.State{CurrentChannelID: 2, Unreads: map[int64]int{}}

	st.ApplyIncomingMessage(model.Message{ID: 10, ChannelID: 3, Content: "hello"})

	if len(st.Messages) != 0 || st.Unreads[3] != 1 {
		t.Fatalf("state = %#v", st)
	}
}

func TestStateRemovesDeletedVisibleMessage(t *testing.T) {
	st := app.State{Messages: []model.Message{{ID: 10}, {ID: 11}}}

	st.ApplyDeletedMessage(10)

	if len(st.Messages) != 1 || st.Messages[0].ID != 11 {
		t.Fatalf("messages = %#v", st.Messages)
	}
}
