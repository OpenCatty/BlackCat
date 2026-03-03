// Package session provides conversation session management with persistent storage.
package session

// SessionKey uniquely identifies a session by channel type, channel ID, and user ID.
type SessionKey struct {
	ChannelType string
	ChannelID   string
	UserID      string // empty for anonymous users
}

// String returns "channelType:channelID:userID".
// For anonymous users (empty UserID), returns "channelType:channelID".
func (k SessionKey) String() string {
	if k.UserID == "" {
		return k.ChannelType + ":" + k.ChannelID
	}
	return k.ChannelType + ":" + k.ChannelID + ":" + k.UserID
}
