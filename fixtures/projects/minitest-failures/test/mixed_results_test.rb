require 'minitest/autorun'

class User
  attr_accessor :name, :age, :email

  def initialize(name, age, email = nil)
    @name = name
    @age = age
    @email = email
  end

  def adult?
    age >= 18
  end

  def valid_email?
    return false if email.nil?
    email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
  end

  def display_name
    name.upcase
  end
end

class MixedResultsTest < Minitest::Test
  def setup
    @user = User.new("John Doe", 25, "john@example.com")
  end

  def test_user_initialization_success
    assert_equal "John Doe", @user.name
    assert_equal 25, @user.age
    assert_equal "john@example.com", @user.email
  end

  def test_adult_check_passes
    assert @user.adult?
    
    minor = User.new("Jane", 16)
    refute minor.adult?
  end

  def test_display_name_failure
    # This will fail - expecting wrong case
    assert_equal "john doe", @user.display_name
  end

  def test_email_validation_mixed
    assert @user.valid_email?
    
    # This will fail - invalid email should return false
    @user.email = "not-an-email"
    assert @user.valid_email?
  end

  def test_nil_handling_error
    user_without_email = User.new("Bob", 30)
    # This will fail - doesn't handle nil properly
    assert_equal "", user_without_email.email
  end

  def test_age_boundary_success
    adult = User.new("Adult", 18)
    assert adult.adult?
  end

  def test_type_error_will_fail
    # This will cause an error - comparing different types
    assert_equal "25", @user.age
  end

  def skip_test_not_implemented_yet
    # This test is skipped
    skip "Feature not implemented yet"
    assert false
  end
end