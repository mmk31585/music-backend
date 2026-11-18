# Integration Test Utilities

This package provides test utilities for integration testing the Muse backend application, including:

## Overview

The integration tests are designed to test the application in a realistic environment with actual database and cache services running. This package provides helpers for setting up and tearing down test container environments.

## Test Containers

The package uses `testcontainers-go` to spin up temporary containers for:

- **PostgreSQL 17**: For database testing
- **Redis 7**: For cache testing

## Usage

### Setting Up Test Environment

```go
import (
	"testing"
	"music/internal/testutil"
)

func TestWithContainers(t *testing.T) {
	tc, err := testutil.NewTestContainer(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer func() {
		if err := tc.Stop(); err != nil {
			t.Logf("failed to stop containers: %v", err)
		}
	}()

	if err := tc.Start(t); err != nil {
		 t.Fatalf("failed to start containers: %v", err)
	}

	// Use test container for your tests
	pgURL := tc.GetPostgresDSN()
	redisAddr := tc.GetRedisAddr()

	// Your test code here...
}
```

### HTTP Request Helpers

For testing HTTP endpoints:

```go
import (
	"music/internal/testutil"
)

func TestAPI(t *testing.T) {
	// Build your gin router first
	router := buildTestRouter(t)

	// Test with token
	resp, err := testutil.ExecuteRequestWithToken(
		router,
		"GET",
		"/api/v1/test",
		"test-token",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
```

## Building Tags

Integration tests should be run with the `integration` build tag:

```bash
go test -tags integration ./internal/testutil/...
```

## CI Configuration

The CI workflow in `.github/workflows/ci.yml` includes an integration test job (currently disabled by default). To enable integration tests:

1. Remove the `if: false` condition from the `test-integration` job
2. Ensure `testcontainers-go` is added as a dependency
3. Update the go.mod file if necessary

## Test Data

For tests that need test data, you can:

1. Use the existing test fixtures in `testutil/fixtures/`
2. Create test data using the test container's database connection
3. Use the existing test data setup from the migrations

## Best Practices

1. **Cleanup**: Always defer container cleanup to ensure resources are released
2. **Error handling**: Proper error handling for container operations
3. **Portability**: Use DSN/Addr methods instead of hardcoded connection strings
4. **Concurrency**: Consider parallel test execution with separate test containers
5. **Speed**: Integration tests are slower than unit tests, so run them less frequently

## Testing Strategy

Use integration tests for:

- End-to-end testing of complex workflows
- Testing with realistic data and constraints
- Testing interactions between multiple services (e.g., API + Database + Cache)
- Performance testing with realistic workloads
- Testing error conditions and edge cases

Use unit tests for:

- Individual functions and methods
- Simple logic testing
- Fast, isolated tests
- Mocking external dependencies

## Limitations

1. Container startup time: Tests may take longer due to container startup
2. Resource usage: Multiple containers require more system resources
3. Complex setup: Integration tests require more setup code
4. Flakiness: Container-based tests can be flaky depending on the environment

## Future Enhancements

1. Add support for more test containers (e.g., OpenSearch)
2. Add fixtures for common test scenarios
3. Add performance measurement utilities
4. Add test data factories for realistic test data