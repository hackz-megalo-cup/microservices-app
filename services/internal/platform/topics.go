package platform

// Kafka topics used across services.
const (
	TopicGreetingCreated   = "greeting.created"
	TopicCallCompleted     = "call.completed"
	TopicInvocationCreated = "invocation.created"
	TopicUserRegistered    = "user.registered"

	// Saga compensation topics.
	TopicGreetingFailed   = "greeting.failed"
	TopicInvocationFailed = "invocation.failed"

	// Dead Letter Queue topics.
	TopicGreetingCreatedDLQ   = "greeting.created.dlq"
	TopicCallCompletedDLQ     = "call.completed.dlq"
	TopicInvocationCreatedDLQ = "invocation.created.dlq"
	TopicUserRegisteredDLQ    = "user.registered.dlq"
)

// DLQTopic returns the dead-letter queue topic for a given source topic.
// Returns empty string if no DLQ mapping exists.
func DLQTopic(source string) string {
	m := map[string]string{
		TopicGreetingCreated:   TopicGreetingCreatedDLQ,
		TopicCallCompleted:     TopicCallCompletedDLQ,
		TopicInvocationCreated: TopicInvocationCreatedDLQ,
		TopicUserRegistered:    TopicUserRegisteredDLQ,
	}
	return m[source]
}

// DefaultTopics returns all topics with their partition counts.
func DefaultTopics() map[string]int32 {
	return map[string]int32{
		TopicGreetingCreated:      3,
		TopicCallCompleted:        3,
		TopicInvocationCreated:    3,
		TopicUserRegistered:       3,
		TopicGreetingFailed:       1,
		TopicInvocationFailed:     1,
		TopicGreetingCreatedDLQ:   1,
		TopicCallCompletedDLQ:     1,
		TopicInvocationCreatedDLQ: 1,
		TopicUserRegisteredDLQ:    1,
	}
}
