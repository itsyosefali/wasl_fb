package events

const (
	MessageReceived         = "message.received"
	MessageSent             = "message.sent"
	CommentCreated          = "comment.created"
	CommentUpdated          = "comment.updated"
	CommentDeleted          = "comment.deleted"
	CommentReplied          = "comment.replied"
	CommentHidden           = "comment.hidden"
	PageConnected           = "page.connected"
	InstagramMessageRecv    = "instagram.message.received"
	InstagramCommentCreated = "instagram.comment.created"

	AggregateMessage = "message"
	AggregateComment = "comment"
	AggregateContact = "contact"
	AggregatePage    = "page"
)

func Subject(tenantID, eventType string) string {
	return "events." + tenantID + "." + eventType
}

func WildcardSubject() string {
	return "events.>"
}

const (
	StreamName    = "GATEWAY_EVENTS"
	ConsumerName  = "gateway-delivery-v2"
	StreamSubject = "events.>"
)
