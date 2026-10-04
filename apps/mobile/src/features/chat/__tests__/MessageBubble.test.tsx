import { fireEvent, render, screen } from "@testing-library/react-native";
import React from "react";
import type { ChatMessage } from "../../../api/types";
import { MessageBubble, statusLabel } from "../MessageBubble";

const base: ChatMessage = {
  id: "1",
  conversationId: "c",
  senderId: "me",
  body: "hello",
  createdAt: "2026-01-01T10:00:00Z",
  readAt: null
};

describe("MessageBubble", () => {
  it("shows Read when readAt is set", () => {
    expect(statusLabel({ ...base, readAt: "2026-01-01T10:05:00Z" })).toMatch(/Read$/);
    expect(statusLabel(base)).not.toMatch(/Read/);
    expect(statusLabel({ ...base, status: "sending" })).toBe("Sending…");
  });

  it("offers retry and discard for failed messages", () => {
    const onRetry = jest.fn();
    const onDiscard = jest.fn();
    const failed = { ...base, status: "failed" as const };
    render(<MessageBubble message={failed} mine onDiscard={onDiscard} onRetry={onRetry} />);
    fireEvent.press(screen.getByRole("button", { name: "Message not sent. Retry" }));
    fireEvent.press(screen.getByRole("button", { name: "Delete unsent message" }));
    expect(onRetry).toHaveBeenCalledWith(failed);
    expect(onDiscard).toHaveBeenCalledWith(failed);
  });
});
