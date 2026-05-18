package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilteredMessagesNoQuery(t *testing.T) {
	m := &AppModel{
		messages: []MessageItem{
			{Text: "hello world"},
			{Text: "goodbye world"},
		},
	}
	filtered, count := m.filteredMessages()
	assert.Equal(t, 2, count)
	assert.Equal(t, m.messages, filtered)
}

func TestFilteredMessagesWithQuery(t *testing.T) {
	m := &AppModel{
		messages: []MessageItem{
			{Text: "hello world"},
			{Text: "goodbye world"},
			{Text: "foo bar"},
		},
		searchQuery: "world",
	}
	filtered, count := m.filteredMessages()
	assert.Equal(t, 2, count)
	assert.Equal(t, "hello world", filtered[0].Text)
	assert.Equal(t, "goodbye world", filtered[1].Text)
}

func TestFilteredMessagesCaseInsensitive(t *testing.T) {
	m := &AppModel{
		messages: []MessageItem{
			{Text: "Hello World"},
			{Text: "nope"},
		},
		searchQuery: "hello",
	}
	filtered, count := m.filteredMessages()
	assert.Equal(t, 1, count)
	assert.Equal(t, "Hello World", filtered[0].Text)
}

func TestFilteredMessagesNoMatches(t *testing.T) {
	m := &AppModel{
		messages: []MessageItem{
			{Text: "hello world"},
		},
		searchQuery: "zzzzz",
	}
	filtered, count := m.filteredMessages()
	assert.Equal(t, 0, count)
	assert.Empty(t, filtered)
}

func TestFilteredMessagesEmptyMessages(t *testing.T) {
	m := &AppModel{
		messages:    nil,
		searchQuery: "test",
	}
	filtered, count := m.filteredMessages()
	assert.Equal(t, 0, count)
	assert.Empty(t, filtered)
}

func TestRebuildMessagesFromBuf(t *testing.T) {
	m := &AppModel{
		msgBufs: map[string][]MessageItem{
			"#lobby": {
				{Text: "original"},
			},
		},
		activeDMConv: "#lobby",
		messages: []MessageItem{
			{Text: "filtered"},
		},
		searchQuery: "test",
	}
	m.rebuildMessagesFromBuf()
	assert.Equal(t, 1, len(m.messages))
	assert.Equal(t, "original", m.messages[0].Text)
}

func TestRebuildMessagesFromBufMissingKey(t *testing.T) {
	m := &AppModel{
		msgBufs:      map[string][]MessageItem{},
		activeDMConv: "#lobby",
		messages: []MessageItem{
			{Text: "old"},
		},
	}
	m.rebuildMessagesFromBuf()
	assert.Empty(t, m.messages)
}

func TestExecSearchSetsQuery(t *testing.T) {
	m := &AppModel{
		msgBufs: map[string][]MessageItem{
			"#lobby": {
				{Text: "hello world"},
				{Text: "goodbye"},
			},
		},
		activeDMConv: "#lobby",
		messages: []MessageItem{
			{Text: "hello world"},
			{Text: "goodbye"},
		},
	}
	m.execSearch([]string{"search", "world"})
	assert.Equal(t, "world", m.searchQuery)
	// messages: 2 restored + 1 system msg (search result, which also contains "world")
	assert.Equal(t, 3, len(m.messages))
	// filtered: "hello world" + system msg "searching for "world"..." = 2
	_, count := m.filteredMessages()
	assert.Equal(t, 2, count)
}

func TestExecSearchClearsWhenAlreadySet(t *testing.T) {
	m := &AppModel{
		msgBufs: map[string][]MessageItem{
			"#lobby": {
				{Text: "hello world"},
			},
		},
		activeDMConv: "#lobby",
		messages: []MessageItem{
			{Text: "hello world"},
		},
		searchQuery: "world",
	}
	m.execSearch([]string{"search"})
	assert.Empty(t, m.searchQuery)
	// messages = 1 restored + 1 system msg ("search cleared")
	assert.Equal(t, 2, len(m.messages))
}

func TestExecSearchNoArgNoActiveSearch(t *testing.T) {
	m := &AppModel{}
	m.execSearch([]string{"search"})
	assert.Empty(t, m.searchQuery)
}

func TestExecSearchTrimsWhitespace(t *testing.T) {
	m := &AppModel{
		msgBufs: map[string][]MessageItem{
			"#lobby": {
				{Text: "hello world"},
			},
		},
		activeDMConv: "#lobby",
		messages: []MessageItem{
			{Text: "hello world"},
		},
	}
	m.execSearch([]string{"search", "  world  "})
	assert.Equal(t, "world", m.searchQuery)
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{
			name:  "empty",
			text:  "",
			width: 10,
			want:  []string{""},
		},
		{
			name:  "shorter than width",
			text:  "hello",
			width: 10,
			want:  []string{"hello"},
		},
		{
			name:  "exact width",
			text:  "1234567890",
			width: 10,
			want:  []string{"1234567890"},
		},
		{
			name:  "wraps at word boundary",
			text:  "hello world foo",
			width: 10,
			want:  []string{"hello", "world foo"},
		},
		{
			name:  "multiple wraps",
			text:  "a b c d e f g h",
			width: 5,
			want:  []string{"a b c", "d e f", "g h"},
		},
		{
			name:  "single word longer than width",
			text:  "superlongword",
			width: 5,
			want:  []string{"superlongword"},
		},
		{
			name:  "zero width",
			text:  "hello",
			width: 0,
			want:  []string{"hello"},
		},
		{
			name:  "negative width",
			text:  "hello",
			width: -1,
			want:  []string{"hello"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractOtherPeer(t *testing.T) {
	tests := []struct {
		name      string
		topicName string
		myPeerID  string
		want      string
	}{
		{
			name:      "peer A is recipient",
			topicName: "alkalyne/dm/QmA/QmB",
			myPeerID:  "QmA",
			want:      "QmB",
		},
		{
			name:      "peer B is recipient",
			topicName: "alkalyne/dm/QmA/QmB",
			myPeerID:  "QmB",
			want:      "QmA",
		},
		{
			name:      "unexpected prefix",
			topicName: "other/prefix",
			myPeerID:  "QmA",
			want:      "",
		},
		{
			name:      "empty prefix",
			topicName: "",
			myPeerID:  "QmA",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOtherPeer(tt.topicName, tt.myPeerID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShortPeerID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"abc", "abc"},
		{"abcdefghijklmnop", "abcdefgh..mnop"},
		{"1234567890ab", "1234567890ab"},
		{"1234567890abc", "12345678..0abc"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := shortPeerID(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddSystemMsg(t *testing.T) {
	m := &AppModel{}

	m.addSystemMsg("hello")
	assert.Equal(t, 1, len(m.messages))
	assert.Equal(t, "hello", m.messages[0].Text)
	assert.Equal(t, "system", m.messages[0].Sender)

	m.addSystemMsg("world")
	assert.Equal(t, 2, len(m.messages))
}

func TestAddSystemMsgEmptyMessages(t *testing.T) {
	m := &AppModel{
		messages: []MessageItem{},
	}
	m.addSystemMsg("first")
	assert.Equal(t, 1, len(m.messages))
}
