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
	TopicUserLoggedIn      = "user.logged_in"
	TopicCaptureCaught     = "capture.caught"

	// Saga compensation topics.
	TopicGreetingFailed        = "greeting.failed"
	TopicInvocationFailed      = "invocation.failed"
	TopicGreetingCompensated   = "greeting.compensated"
	TopicInvocationCompensated = "invocation.compensated"

	TopicTodoTitleUpdated = "todo.title_updated"
	TopicTodoDeleted      = "todo.deleted"

	TopicItemCreated           = "item.created"
	TopicItemFailed            = "item.failed"
	TopicItemCompensated       = "item.compensated"
	TopicMasterdataCreated     = "masterdata.created"
	TopicMasterdataFailed      = "masterdata.failed"
	TopicMasterdataCompensated = "masterdata.compensated"

	// Dead Letter Queue topics.
	TopicGreetingCreatedDLQ       = "greeting.created.dlq"
	TopicCallCompletedDLQ         = "call.completed.dlq"
	TopicInvocationCreatedDLQ     = "invocation.created.dlq"
	TopicUserRegisteredDLQ        = "user.registered.dlq"
	TopicUserLoggedInDLQ          = "user.logged_in.dlq"
	TopicCaptureCaughtDLQ         = "capture.caught.dlq"
	TopicInvocationCompensatedDLQ = "invocation.compensated.dlq"
	TopicItemCreatedDLQ           = "item.created.dlq"
	TopicMasterdataCreatedDLQ     = "masterdata.created.dlq"
)

// DLQTopic returns the dead-letter queue topic for a given source topic.
// Returns empty string if no DLQ mapping exists.
func DLQTopic(source string) string {
	m := map[string]string{
		TopicGreetingCreated:       TopicGreetingCreatedDLQ,
		TopicCallCompleted:         TopicCallCompletedDLQ,
		TopicInvocationCreated:     TopicInvocationCreatedDLQ,
		TopicUserRegistered:        TopicUserRegisteredDLQ,
		TopicUserLoggedIn:          TopicUserLoggedInDLQ,
		TopicCaptureCaught:         TopicCaptureCaughtDLQ,
		TopicInvocationCompensated: TopicInvocationCompensatedDLQ,
		TopicItemCreated:           TopicItemCreatedDLQ,
		TopicMasterdataCreated:     TopicMasterdataCreatedDLQ,
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
		TopicUserLoggedIn:             3,
		TopicCaptureCaught:            3,
		TopicGreetingFailed:           1,
		TopicInvocationFailed:         1,
		TopicGreetingCompensated:      1,
		TopicInvocationCompensated:    1,
		TopicGreetingCreatedDLQ:       1,
		TopicCallCompletedDLQ:         1,
		TopicInvocationCreatedDLQ:     1,
		TopicUserRegisteredDLQ:        1,
		TopicUserLoggedInDLQ:          1,
		TopicCaptureCaughtDLQ:         1,
		TopicInvocationCompensatedDLQ: 1,
		TopicTodoTitleUpdated:         1,
		TopicTodoDeleted:              1,
		TopicItemCreated:              3,
		TopicItemFailed:               1,
		TopicItemCompensated:          1,
		TopicItemCreatedDLQ:           1,
		TopicMasterdataCreated:        3,
		TopicMasterdataFailed:         1,
		TopicMasterdataCompensated:    1,
		TopicMasterdataCreatedDLQ:     1,
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
