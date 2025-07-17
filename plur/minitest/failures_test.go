package minitest

import (
	"testing"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractFailures_FirstFailure(t *testing.T) {
	assert := assert.New(t)

	input := `  1) Failure:
ArrayOperationsTest#test_average_calculation_failure [test/array_operations_test.rb:33]:
Expected: 3
  Actual: 2.5`

	notifications := ExtractFailures(input)

	assert.Len(notifications, 1)

	notification := notifications[0]
	assert.Equal(types.TestFailed, notification.Event)
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", notification.TestID)
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", notification.Description)
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", notification.FullDescription)
	assert.Equal("test/array_operations_test.rb:33", notification.Location)
	assert.Equal("test/array_operations_test.rb", notification.FilePath)
	assert.Equal(33, notification.LineNumber)
	assert.Equal("failed", notification.Status)
	assert.NotNil(notification.Exception)
	assert.Equal("Minitest::Assertion", notification.Exception.Class)
	assert.Equal("Expected: 3\n  Actual: 2.5", notification.Exception.Message)
	assert.Empty(notification.Exception.Backtrace)
}

func TestExtractFailures_ErrorWithBacktrace(t *testing.T) {
	assert := assert.New(t)

	input := `  3) Error:
ArrayOperationsTest#test_find_max_with_nil:
ArgumentError: comparison of Integer with nil failed
    test/array_operations_test.rb:14:in 'Array#max'
    test/array_operations_test.rb:14:in 'ArrayOperations.find_max'
    test/array_operations_test.rb:38:in 'ArrayOperationsTest#test_find_max_with_nil'`

	notifications := ExtractFailures(input)

	assert.Len(notifications, 1)

	notification := notifications[0]
	assert.Equal(types.TestFailed, notification.Event)
	assert.Equal("ArrayOperationsTest#test_find_max_with_nil", notification.TestID)
	assert.Equal("ArrayOperationsTest#test_find_max_with_nil", notification.Description)
	assert.Equal("test/array_operations_test.rb:38", notification.Location) // Extracted from backtrace
	assert.Equal("test/array_operations_test.rb", notification.FilePath)
	assert.Equal(38, notification.LineNumber)
	assert.Equal("failed", notification.Status)
	assert.NotNil(notification.Exception)
	assert.Equal("ArgumentError", notification.Exception.Class)
	assert.Equal("ArgumentError: comparison of Integer with nil failed", notification.Exception.Message)
	assert.Len(notification.Exception.Backtrace, 3)
	assert.Equal("test/array_operations_test.rb:14:in 'Array#max'", notification.Exception.Backtrace[0])
	assert.Equal("test/array_operations_test.rb:14:in 'ArrayOperations.find_max'", notification.Exception.Backtrace[1])
	assert.Equal("test/array_operations_test.rb:38:in 'ArrayOperationsTest#test_find_max_with_nil'", notification.Exception.Backtrace[2])
}

func TestExtractFailures_ErrorWithoutTestFile(t *testing.T) {
	assert := assert.New(t)

	input := `  1) Error:
DatabaseTest#test_connection_error:
NoMethodError: undefined method 'connect' for nil:NilClass
    lib/database.rb:42:in 'Database#initialize'
    lib/database.rb:15:in 'new'
    lib/application.rb:8:in 'Application#setup'`

	notifications := ExtractFailures(input)

	assert.Len(notifications, 1)

	notification := notifications[0]
	assert.Equal(types.TestFailed, notification.Event)
	assert.Equal("DatabaseTest#test_connection_error", notification.TestID)
	assert.Equal("", notification.Location) // No test file in backtrace
	assert.Equal("", notification.FilePath)
	assert.Equal(0, notification.LineNumber)
	assert.NotNil(notification.Exception)
	assert.Equal("NoMethodError", notification.Exception.Class)
	assert.Equal("NoMethodError: undefined method 'connect' for nil:NilClass", notification.Exception.Message)
	assert.Len(notification.Exception.Backtrace, 3)
	assert.Equal("lib/database.rb:42:in 'Database#initialize'", notification.Exception.Backtrace[0])
	assert.Equal("lib/database.rb:15:in 'new'", notification.Exception.Backtrace[1])
	assert.Equal("lib/application.rb:8:in 'Application#setup'", notification.Exception.Backtrace[2])
}

func TestExtractFailures_ErrorWithTestPrefixFile(t *testing.T) {
	assert := assert.New(t)

	input := `  1) Error:
TestDatabase#test_transaction_rollback:
ActiveRecord::StatementInvalid: PG::ConnectionBad: connection is closed
    lib/database.rb:42:in 'execute'
    lib/database.rb:15:in 'transaction'
    test/test_database.rb:28:in 'block in test_transaction_rollback'`

	notifications := ExtractFailures(input)

	assert.Len(notifications, 1)

	notification := notifications[0]
	assert.Equal(types.TestFailed, notification.Event)
	assert.Equal("TestDatabase#test_transaction_rollback", notification.TestID)
	assert.Equal("test/test_database.rb:28", notification.Location) // Extracted from test_*.rb file
	assert.Equal("test/test_database.rb", notification.FilePath)
	assert.Equal(28, notification.LineNumber)
	assert.NotNil(notification.Exception)
	assert.Equal("ActiveRecord::StatementInvalid", notification.Exception.Class)
	assert.Equal("ActiveRecord::StatementInvalid: PG::ConnectionBad: connection is closed", notification.Exception.Message)
	assert.Len(notification.Exception.Backtrace, 3)
}

func TestExtractFailures_CompleteExample(t *testing.T) {
	assert := assert.New(t)

	input := `  1) Failure:
ArrayOperationsTest#test_average_calculation_failure [test/array_operations_test.rb:33]:
Expected: 3
  Actual: 2.5

  2) Failure:
ArrayOperationsTest#test_average_precision_failure [test/array_operations_test.rb:47]:
Expected: 2.33
  Actual: 2.3333333333333335

  3) Error:
ArrayOperationsTest#test_find_max_with_nil:
ArgumentError: comparison of Integer with nil failed
    test/array_operations_test.rb:14:in 'Array#max'
    test/array_operations_test.rb:14:in 'ArrayOperations.find_max'
    test/array_operations_test.rb:38:in 'ArrayOperationsTest#test_find_max_with_nil'`

	notifications := ExtractFailures(input)

	assert.Len(notifications, 3)

	// First failure
	assert.Equal(types.TestFailed, notifications[0].Event)
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", notifications[0].TestID)
	assert.Equal("test/array_operations_test.rb:33", notifications[0].Location)
	assert.Equal("Minitest::Assertion", notifications[0].Exception.Class)
	assert.Equal("Expected: 3\n  Actual: 2.5", notifications[0].Exception.Message)

	// Second failure
	assert.Equal(types.TestFailed, notifications[1].Event)
	assert.Equal("ArrayOperationsTest#test_average_precision_failure", notifications[1].TestID)
	assert.Equal("test/array_operations_test.rb:47", notifications[1].Location)
	assert.Equal("Minitest::Assertion", notifications[1].Exception.Class)
	assert.Equal("Expected: 2.33\n  Actual: 2.3333333333333335", notifications[1].Exception.Message)

	// Third error
	assert.Equal(types.TestFailed, notifications[2].Event)
	assert.Equal("ArrayOperationsTest#test_find_max_with_nil", notifications[2].TestID)
	assert.Equal("test/array_operations_test.rb:38", notifications[2].Location)
	assert.Equal("ArgumentError", notifications[2].Exception.Class)
	assert.Equal("ArgumentError: comparison of Integer with nil failed", notifications[2].Exception.Message)
	assert.Len(notifications[2].Exception.Backtrace, 3)
}
