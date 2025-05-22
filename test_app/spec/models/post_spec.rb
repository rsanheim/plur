require 'rails_helper'

RSpec.describe Post, type: :model do
  let(:user) { User.create!(name: "John Doe", email: "john@example.com") }

  it "creates a post with valid attributes" do
    post = Post.create!(title: "Test Post", content: "Test content", user: user)
    expect(post.title).to eq("Test Post")
    expect(post.content).to eq("Test content")
    expect(post.user).to eq(user)
  end

  it "requires a title" do
    post = Post.new(content: "Test content", user: user)
    expect(post).not_to be_valid
    expect(post.errors[:title]).to include("can't be blank")
  end

  it "requires content" do
    post = Post.new(title: "Test Post", user: user)
    expect(post).not_to be_valid
    expect(post.errors[:content]).to include("can't be blank")
  end

  it "requires a user" do
    post = Post.new(title: "Test Post", content: "Test content")
    expect(post).not_to be_valid
    expect(post.errors[:user]).to include("must exist")
  end

  it "prints environment info for parallel testing" do
    puts "Post spec - TEST_ENV_NUMBER: '#{ENV['TEST_ENV_NUMBER']}'"
    puts "Post spec - PARALLEL_TEST_GROUPS: '#{ENV['PARALLEL_TEST_GROUPS']}'"
    expect(true).to be true
  end
end
