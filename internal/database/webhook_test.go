package database

// Webhook database integration tests.
// These tests require a running PostgreSQL instance and are skipped in CI
// without the RAMPART_TEST_DATABASE_URL environment variable.
//
// To run locally:
//   RAMPART_TEST_DATABASE_URL="postgres://rampart:rampart@localhost:5432/rampart_test" go test ./internal/database/ -run TestWebhook
//
// Tests to implement:
// - TestWebhookCreateAndGet: insert a webhook, retrieve by ID, verify fields
// - TestWebhookList: insert multiple webhooks, list by org
// - TestWebhookUpdate: update URL/events/enabled, verify changes
// - TestWebhookDelete: delete webhook, verify gone
// - TestWebhooksForEvent: insert webhooks with different events, query by event type
// - TestWebhookDeliveryCreate: insert delivery, verify fields
// - TestWebhookDeliveryList: insert deliveries, list by webhook ID
// - TestPendingRetries: insert deliveries with next_retry_at, query pending
