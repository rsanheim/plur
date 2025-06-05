require "rails_helper"

RSpec.describe User, type: :model do
  it "creates a user with valid attributes" do
    user = User.create!(name: "John Doe", email: "john@example.com")
    expect(user.name).to eq("John Doe")
    expect(user.email).to eq("john@example.com")
  end

  it "requires a name" do
    user = User.new(email: "john@example.com")
    expect(user).not_to be_valid
    expect(user.errors[:name]).to include("can't be blank")
  end

  it "requires an email" do
    user = User.new(name: "John Doe")
    expect(user).not_to be_valid
    expect(user.errors[:email]).to include("can't be blank")
  end

  it "requires unique email" do
    User.create!(name: "John Doe", email: "john@example.com")
    duplicate_user = User.new(name: "Jane Doe", email: "john@example.com")
    expect(duplicate_user).not_to be_valid
    expect(duplicate_user.errors[:email]).to include("has already been taken")
  end

  it "can have many posts" do
    user = User.create!(name: "John Doe", email: "john@example.com")
    post1 = user.posts.create!(title: "First Post", content: "Hello World")
    post2 = user.posts.create!(title: "Second Post", content: "Goodbye World")

    expect(user.posts.count).to eq(2)
    expect(user.posts).to include(post1, post2)
  end

  it "prints environment info for parallel testing" do
    puts "User spec - TEST_ENV_NUMBER: '#{ENV["TEST_ENV_NUMBER"]}'"
    puts "User spec - PARALLEL_TEST_GROUPS: '#{ENV["PARALLEL_TEST_GROUPS"]}'"
    expect(true).to be true
  end
end
