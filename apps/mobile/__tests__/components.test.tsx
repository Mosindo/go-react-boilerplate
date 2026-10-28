import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { AuthForm } from "../src/components/AuthForm";
import { ConversationList } from "../src/components/ConversationList";
import type { Conversation } from "../src/api/types";

jest.mock("react-native-safe-area-context", () => ({
  SafeAreaView: ({ children }: { children: React.ReactNode }) => children,
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 })
}));

describe("AuthForm", () => {
  it("shows validation errors and does not submit invalid input", () => {
    const onSubmit = jest.fn();
    render(<AuthForm submitLabel="Continue" strictPassword onSubmit={onSubmit} />);
    fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(screen.getByText("Enter your email address.")).toBeTruthy();
    expect(screen.getByText("Use at least 8 characters.")).toBeTruthy();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits trimmed credentials once valid", () => {
    const onSubmit = jest.fn();
    render(<AuthForm submitLabel="Continue" strictPassword onSubmit={onSubmit} />);
    fireEvent.changeText(screen.getByTestId("auth-email-input"), " me@example.com ");
    fireEvent.changeText(screen.getByTestId("auth-password-input"), "correct horse");
    fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(onSubmit).toHaveBeenCalledWith("me@example.com", "correct horse");
  });

  it("renders a server error", () => {
    render(<AuthForm submitLabel="Sign in" error="invalid credentials" onSubmit={jest.fn()} />);
    expect(screen.getByTestId("auth-error")).toBeTruthy();
  });
});

describe("ConversationList", () => {
  it("shows an empty state", () => {
    render(<ConversationList conversations={[]} myUserId="me" onOpen={jest.fn()} />);
    expect(screen.getByText("No conversations yet")).toBeTruthy();
  });

  it("renders previews, unread badges and opens a conversation", () => {
    const conversation: Conversation = {
      id: "c1",
      matchedAt: "2026-01-01T00:00:00Z",
      user: { userId: "u2", firstName: "Sam", age: 29, photoUrl: null },
      lastMessage: { id: "m1", conversationId: "c1", senderId: "u2", body: "Hello there", createdAt: "2026-01-01T10:00:00Z", readAt: null },
      unreadCount: 2
    };
    const onOpen = jest.fn();
    render(<ConversationList conversations={[conversation]} myUserId="me" onOpen={onOpen} />);
    expect(screen.getByText("Hello there")).toBeTruthy();
    expect(screen.getByLabelText("2 unread")).toBeTruthy();
    fireEvent.press(screen.getByTestId("conversation-c1"));
    expect(onOpen).toHaveBeenCalledWith(conversation);
  });
});
