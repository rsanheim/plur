require 'test/unit'

class InsufficientFundsError < StandardError; end

class BankAccount
  attr_reader :balance, :account_number

  def initialize(account_number, initial_balance = 0)
    @account_number = account_number
    @balance = initial_balance
  end

  def deposit(amount)
    raise ArgumentError, "Amount must be positive" if amount <= 0
    @balance += amount
  end

  def withdraw(amount)
    raise ArgumentError, "Amount must be positive" if amount <= 0
    raise InsufficientFundsError, "Insufficient funds" if amount > @balance
    @balance -= amount
  end

  def transfer_to(other_account, amount)
    withdraw(amount)
    other_account.deposit(amount)
  end
end

class TestBankAccount < Test::Unit::TestCase
  def setup
    @account = BankAccount.new("12345", 1000)
    @other_account = BankAccount.new("67890", 500)
  end

  def test_initialization_success
    assert_equal "12345", @account.account_number
    assert_equal 1000, @account.balance
  end

  def test_deposit_success
    @account.deposit(500)
    assert_equal 1500, @account.balance
  end

  def test_deposit_negative_amount_failure
    # This will pass - correctly raises error
    assert_raise(ArgumentError) { @account.deposit(-100) }
  end

  def test_withdraw_success
    @account.withdraw(300)
    assert_equal 700, @account.balance
  end

  def test_withdraw_insufficient_funds_wrong_error
    # This will fail - expecting wrong error type
    assert_raise(ArgumentError) { @account.withdraw(2000) }
  end

  def test_balance_after_multiple_operations_failure
    @account.deposit(500)
    @account.withdraw(200)
    # This will fail - wrong expected value
    assert_equal 1200, @account.balance
  end

  def test_transfer_success
    @account.transfer_to(@other_account, 200)
    assert_equal 800, @account.balance
    assert_equal 700, @other_account.balance
  end

  def test_transfer_with_insufficient_funds_no_error_handling
    # This will raise an error that's not caught
    @account.transfer_to(@other_account, 2000)
    assert_equal 1000, @account.balance
  end

  def test_float_precision_failure
    @account.deposit(0.1)
    @account.deposit(0.2)
    # This might fail due to float precision
    assert_equal 1000.3, @account.balance
  end

  def test_pending_feature
    omit("Interest calculation not implemented yet")
    assert_equal 1050, @account.calculate_interest(0.05)
  end
end