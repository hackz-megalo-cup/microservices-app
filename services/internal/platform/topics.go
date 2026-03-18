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

	// Saga compensation topics.
	TopicGreetingFailed        = "greeting.failed"
	TopicInvocationFailed      = "invocation.failed"
	TopicGreetingCompensated   = "greeting.compensated"
	TopicInvocationCompensated = "invocation.compensated"
	TopicUserFailed            = "user.failed"
	TopicUserCompensated       = "user.compensated"

	TopicTodoTitleUpdated = "todo.title_updated"
	TopicTodoDeleted      = "todo.deleted"

	TopicItemCreated           = "item.created"
	TopicItemFailed            = "item.failed"
	TopicItemGranted           = "item.granted"
	TopicItemUsed              = "item.used"
	TopicItemCompensated       = "item.compensated"
	TopicMasterdataCreated     = "masterdata.created"
	TopicMasterdataFailed      = "masterdata.failed"
	TopicMasterdataCompensated = "masterdata.compensated"

	TopicRaidLobbyCreated     = "raid_lobby.created"
	TopicRaidLobbyFinished    = "raid_lobby.finished"
	TopicRaidLobbyFailed      = "raid_lobby.failed"
	TopicRaidLobbyCompensated = "raid_lobby.compensated"
	TopicRaidUserJoined       = "raid.user_joined"
	TopicRaidBattleStarted    = "raid.battle_started"

	TopicBattleFinished = "battle.finished"

	TopicLobbyCreated     = "lobby.created"
	TopicLobbyFailed      = "lobby.failed"
	TopicLobbyCompensated = "lobby.compensated"

	TopicCaptureStarted     = "capture.started"
	TopicCaptureItemUsed    = "capture.item_used"
	TopicCaptureBallThrown  = "capture.ball_thrown"
	TopicCaptureCompleted   = "capture.completed"
	TopicCaptureFailed      = "capture.failed"
	TopicCaptureCompensated = "capture.compensated"

	// Dead Letter Queue topics.
	TopicGreetingCreatedDLQ       = "greeting.created.dlq"
	TopicCallCompletedDLQ         = "call.completed.dlq"
	TopicInvocationCreatedDLQ     = "invocation.created.dlq"
	TopicUserRegisteredDLQ        = "user.registered.dlq"
	TopicUserLoggedInDLQ          = "user.logged_in.dlq"
	TopicUserFailedDLQ            = "user.failed.dlq"
	TopicUserCompensatedDLQ       = "user.compensated.dlq"
	TopicInvocationCompensatedDLQ = "invocation.compensated.dlq"
	TopicItemCreatedDLQ           = "item.created.dlq"
	TopicMasterdataCreatedDLQ     = "masterdata.created.dlq"
	TopicRaidLobbyCreatedDLQ      = "raid_lobby.created.dlq"
	TopicRaidUserJoinedDLQ        = "raid.user_joined.dlq"
	TopicRaidBattleStartedDLQ     = "raid.battle_started.dlq"
	TopicLobbyCreatedDLQ          = "lobby.created.dlq"
	TopicCaptureStartedDLQ        = "capture.started.dlq"
	TopicCaptureCompletedDLQ      = "capture.completed.dlq"
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
		TopicUserFailed:            TopicUserFailedDLQ,
		TopicUserCompensated:       TopicUserCompensatedDLQ,
		TopicCaptureCompleted:      TopicCaptureCompletedDLQ,
		TopicInvocationCompensated: TopicInvocationCompensatedDLQ,
		TopicItemCreated:           TopicItemCreatedDLQ,
		TopicMasterdataCreated:     TopicMasterdataCreatedDLQ,
		TopicRaidLobbyCreated:      TopicRaidLobbyCreatedDLQ,
		TopicRaidUserJoined:        TopicRaidUserJoinedDLQ,
		TopicRaidBattleStarted:     TopicRaidBattleStartedDLQ,
		TopicLobbyCreated:          TopicLobbyCreatedDLQ,
		TopicCaptureStarted:        TopicCaptureStartedDLQ,
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
		TopicCaptureCompleted:         3,
		TopicGreetingFailed:           1,
		TopicInvocationFailed:         1,
		TopicGreetingCompensated:      1,
		TopicInvocationCompensated:    1,
		TopicUserFailed:               1,
		TopicUserCompensated:          1,
		TopicGreetingCreatedDLQ:       1,
		TopicCallCompletedDLQ:         1,
		TopicInvocationCreatedDLQ:     1,
		TopicUserRegisteredDLQ:        1,
		TopicUserLoggedInDLQ:          1,
		TopicUserFailedDLQ:            1,
		TopicUserCompensatedDLQ:       1,
		TopicCaptureCompletedDLQ:      1,
		TopicInvocationCompensatedDLQ: 1,
		TopicTodoTitleUpdated:         1,
		TopicTodoDeleted:              1,
		TopicItemCreated:              3,
		TopicItemFailed:               1,
		TopicItemGranted:              3,
		TopicItemUsed:                 3,
		TopicItemCompensated:          1,
		TopicItemCreatedDLQ:           1,
		TopicMasterdataCreated:        3,
		TopicMasterdataFailed:         1,
		TopicMasterdataCompensated:    1,
		TopicMasterdataCreatedDLQ:     1,
		TopicRaidLobbyCreated:         3,
		TopicRaidLobbyFinished:        3,
		TopicRaidLobbyFailed:          1,
		TopicRaidLobbyCompensated:     1,
		TopicRaidLobbyCreatedDLQ:      1,
		TopicRaidUserJoined:           3,
		TopicRaidUserJoinedDLQ:        1,
		TopicRaidBattleStarted:        3,
		TopicRaidBattleStartedDLQ:     1,
		TopicBattleFinished:           3,
		TopicLobbyCreated:             3,
		TopicLobbyFailed:              1,
		TopicLobbyCompensated:         1,
		TopicLobbyCreatedDLQ:          1,
		TopicCaptureStarted:           3,
		TopicCaptureItemUsed:          3,
		TopicCaptureBallThrown:        3,
		TopicCaptureFailed:            1,
		TopicCaptureCompensated:       1,
		TopicCaptureStartedDLQ:        1,
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
