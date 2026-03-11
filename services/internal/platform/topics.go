package platform

import (
	"sort"
	"strings"
)

// Kafka topics used across services.
const (
	TopicGreetingCreated   = "greeting.created"
	TopicCallCompleted     = "call.completed"
	TopicInvocationCreated = "invocation.created"
	TopicUserRegistered    = "user.registered"

	// Saga compensation topics.
	TopicGreetingFailed        = "greeting.failed"
	TopicInvocationFailed      = "invocation.failed"
	TopicGreetingCompensated   = "greeting.compensated"
	TopicInvocationCompensated = "invocation.compensated"

	TopicTodoTitleUpdated = "todo.title_updated"
	TopicTodoDeleted      = "todo.deleted"

	// Dead Letter Queue topics.
	TopicGreetingCreatedDLQ       = "greeting.created.dlq"
	TopicCallCompletedDLQ         = "call.completed.dlq"
	TopicInvocationCreatedDLQ     = "invocation.created.dlq"
	TopicUserRegisteredDLQ        = "user.registered.dlq"
	TopicInvocationCompensatedDLQ = "invocation.compensated.dlq"
)

// DLQTopic returns the dead-letter queue topic for a given source topic.
// Returns empty string if no DLQ mapping exists.
func DLQTopic(source string) string {
	m := map[string]string{
		TopicGreetingCreated:       TopicGreetingCreatedDLQ,
		TopicCallCompleted:         TopicCallCompletedDLQ,
		TopicInvocationCreated:     TopicInvocationCreatedDLQ,
		TopicUserRegistered:        TopicUserRegisteredDLQ,
		TopicInvocationCompensated: TopicInvocationCompensatedDLQ,
	}
	return m[source]
}

// DefaultTopics returns all topics with their partition counts.
func DefaultTopics() map[string]int32 {
	return map[string]int32{
		TopicGreetingCreated:          3,
		TopicCallCompleted:            3,
		TopicInvocationCreated:        3,
		TopicUserRegistered:           3,
		TopicGreetingFailed:           1,
		TopicInvocationFailed:         1,
		TopicGreetingCompensated:      1,
		TopicInvocationCompensated:    1,
		TopicGreetingCreatedDLQ:       1,
		TopicCallCompletedDLQ:         1,
		TopicInvocationCreatedDLQ:     1,
		TopicUserRegisteredDLQ:        1,
		TopicInvocationCompensatedDLQ: 1,
		TopicTodoTitleUpdated:         1,
		TopicTodoDeleted:              1,
	}
}

// SubscribableTopics returns all non-DLQ topics sorted alphabetically.
func SubscribableTopics() []string {
	var topics []string
	for topic := range DefaultTopics() {
		if !strings.HasSuffix(topic, ".dlq") {
			topics = append(topics, topic)
		}
	}
	sort.Strings(topics)
	return topics
}
