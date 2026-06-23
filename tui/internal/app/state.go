package app

import "hive-tui/internal/model"

type State struct {
	CurrentChannelID int64
	Channels         []model.Channel
	Messages         []model.Message
	Unreads          map[int64]int
}

func (s *State) SelectChannel(channelID int64) {
	s.CurrentChannelID = channelID
	s.Messages = nil
	s.ensureUnreads()
	s.Unreads[channelID] = 0
}

func (s *State) ApplyIncomingMessage(message model.Message) {
	s.ensureUnreads()
	if message.ChannelID == s.CurrentChannelID {
		s.Messages = append(s.Messages, message)
		return
	}
	s.Unreads[message.ChannelID]++
}

func (s *State) ApplyDeletedMessage(messageID int64) {
	next := s.Messages[:0]
	for _, message := range s.Messages {
		if message.ID != messageID {
			next = append(next, message)
		}
	}
	s.Messages = next
}

func (s *State) ensureUnreads() {
	if s.Unreads == nil {
		s.Unreads = map[int64]int{}
	}
}
