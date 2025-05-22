require "spec_helper"

class User
  attr_accessor :name, :email
  
  def initialize(name, email)
    @name = name
    @email = email
  end
  
  def display_name
    "#{name} (#{email})"
  end
end

RSpec.describe User do
  let(:user) { User.new("John Doe", "john@example.com") }

  describe "#display_name" do
    it "combines name and email" do
      expect(user.display_name).to eq("John Doe (john@example.com)")
    end
  end

  describe "#name" do
    it "returns the user's name" do
      expect(user.name).to eq("John Doe")
    end
  end

  describe "#email" do
    it "returns the user's email" do
      expect(user.email).to eq("john@example.com")
    end
  end
end