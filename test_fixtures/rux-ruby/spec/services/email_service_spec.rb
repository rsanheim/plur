require "spec_helper"

class EmailService
  def self.send_email(to, subject, body)
    # Simulate sending email
    {status: "sent", to: to, subject: subject}
  end

  def self.validate_email(email)
    email.include?("@") && email.include?(".")
  end
end

RSpec.describe EmailService do
  describe ".send_email" do
    it "returns sent status" do
      result = EmailService.send_email("test@example.com", "Hello", "Test message")
      expect(result[:status]).to eq("sent")
    end

    it "includes recipient in result" do
      result = EmailService.send_email("test@example.com", "Hello", "Test message")
      expect(result[:to]).to eq("test@example.com")
    end
  end

  describe ".validate_email" do
    it "validates correct email format" do
      expect(EmailService.validate_email("test@example.com")).to be true
    end

    it "rejects invalid email format" do
      expect(EmailService.validate_email("invalid-email")).to be false
    end
  end
end
