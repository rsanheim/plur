package minitest

import (
	"testing"

	"github.com/rsanheim/rux/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractFailures_FirstFailure(t *testing.T) {
	assert := assert.New(t)
	
	input := `  1) Failure:
ArrayOperationsTest#test_average_calculation_failure [test/array_operations_test.rb:33]:
Expected: 3
  Actual: 2.5`

	failures := ExtractFailures(input)

	assert.Len(failures, 1)

	failure := failures[0]
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", failure.Description)
	assert.Equal("test/array_operations_test.rb:33", failure.Location)
	assert.Equal("test/array_operations_test.rb", failure.FilePath)
	assert.Equal(33, failure.LineNumber)
	assert.Equal("Expected: 3\n  Actual: 2.5", failure.Message)
	assert.Equal(types.TestState("failure"), failure.State)
}

func TestExtractFailures_ErrorWithBacktrace(t *testing.T) {
	assert := assert.New(t)
	
	input := `  3) Error:
ArrayOperationsTest#test_find_max_with_nil:
ArgumentError: comparison of Integer with nil failed
    test/array_operations_test.rb:14:in 'Array#max'
    test/array_operations_test.rb:14:in 'ArrayOperations.find_max'
    test/array_operations_test.rb:38:in 'ArrayOperationsTest#test_find_max_with_nil'`

	failures := ExtractFailures(input)

	assert.Len(failures, 1)

	error := failures[0]
	assert.Equal("ArrayOperationsTest#test_find_max_with_nil", error.Description)
	assert.Equal("test/array_operations_test.rb:38", error.Location) // Extracted from backtrace
	assert.Equal("test/array_operations_test.rb", error.FilePath)
	assert.Equal(38, error.LineNumber)
	assert.Equal("ArgumentError: comparison of Integer with nil failed", error.Message)
	assert.Equal(types.TestState("error"), error.State)
	assert.Len(error.Backtrace, 3)
	assert.Equal("test/array_operations_test.rb:14:in 'Array#max'", error.Backtrace[0])
	assert.Equal("test/array_operations_test.rb:14:in 'ArrayOperations.find_max'", error.Backtrace[1])
	assert.Equal("test/array_operations_test.rb:38:in 'ArrayOperationsTest#test_find_max_with_nil'", error.Backtrace[2])
}

func TestExtractFailures_ErrorWithoutTestFile(t *testing.T) {
	assert := assert.New(t)
	
	input := `  1) Error:
DatabaseTest#test_connection_error:
NoMethodError: undefined method 'connect' for nil:NilClass
    lib/database.rb:42:in 'Database#initialize'
    lib/database.rb:15:in 'new'
    lib/application.rb:8:in 'Application#setup'`

	failures := ExtractFailures(input)

	assert.Len(failures, 1)

	error := failures[0]
	assert.Equal("DatabaseTest#test_connection_error", error.Description)
	assert.Equal("", error.Location) // No test file in backtrace
	assert.Equal("", error.FilePath)
	assert.Equal(0, error.LineNumber)
	assert.Equal("NoMethodError: undefined method 'connect' for nil:NilClass", error.Message)
	assert.Equal(types.TestState("error"), error.State)
	assert.Len(error.Backtrace, 3)
	assert.Equal("lib/database.rb:42:in 'Database#initialize'", error.Backtrace[0])
	assert.Equal("lib/database.rb:15:in 'new'", error.Backtrace[1])
	assert.Equal("lib/application.rb:8:in 'Application#setup'", error.Backtrace[2])
}

func TestExtractFailures_ErrorWithTestPrefixFile(t *testing.T) {
	assert := assert.New(t)
	
	input := `  1) Error:
TestDatabase#test_transaction_rollback:
ActiveRecord::StatementInvalid: PG::ConnectionBad: connection is closed
    lib/database.rb:42:in 'execute'
    lib/database.rb:15:in 'transaction'
    test/test_database.rb:28:in 'block in test_transaction_rollback'`

	failures := ExtractFailures(input)

	assert.Len(failures, 1)

	error := failures[0]
	assert.Equal("TestDatabase#test_transaction_rollback", error.Description)
	assert.Equal("test/test_database.rb:28", error.Location) // Extracted from test_*.rb file
	assert.Equal("test/test_database.rb", error.FilePath)
	assert.Equal(28, error.LineNumber)
	assert.Equal("ActiveRecord::StatementInvalid: PG::ConnectionBad: connection is closed", error.Message)
	assert.Equal(types.TestState("error"), error.State)
	assert.Len(error.Backtrace, 3)
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

	failures := ExtractFailures(input)

	assert.Len(failures, 3)

	// First failure
	assert.Equal("ArrayOperationsTest#test_average_calculation_failure", failures[0].Description)
	assert.Equal("test/array_operations_test.rb:33", failures[0].Location)
	assert.Equal(types.TestState("failure"), failures[0].State)
	assert.Equal("Expected: 3\n  Actual: 2.5", failures[0].Message)

	// Second failure
	assert.Equal("ArrayOperationsTest#test_average_precision_failure", failures[1].Description)
	assert.Equal("test/array_operations_test.rb:47", failures[1].Location)
	assert.Equal(types.TestState("failure"), failures[1].State)
	assert.Equal("Expected: 2.33\n  Actual: 2.3333333333333335", failures[1].Message)

	// Third error
	assert.Equal("ArrayOperationsTest#test_find_max_with_nil", failures[2].Description)
	assert.Equal("test/array_operations_test.rb:38", failures[2].Location)
	assert.Equal(types.TestState("error"), failures[2].State)
	assert.Equal("ArgumentError: comparison of Integer with nil failed", failures[2].Message)
	assert.Len(failures[2].Backtrace, 3)
}
