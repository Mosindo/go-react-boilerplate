import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import type { ChatMessage, Interest } from "../../api/types";
import { MessageItem } from "../../shared/ui/MessageItem";
import { Button } from "../../shared/ui/Button";
import { InterestPicker } from "../InterestPicker";

const base: ChatMessage = {
  id: "m1",
  conversationId: "c1",
  senderId: "me",
  body: "See you at 8",
  createdAt: new Date(2026, 0, 1, 20, 5).toISOString(),
  readAt: null
};

describe("Button", () => {
  it("is an accessible button that fires onPress", () => {
    const onPress = jest.fn();
    render(<Button label="Continue" onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Continue" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("ignores presses while loading", () => {
    const onPress = jest.fn();
    render(<Button label="Save" loading onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Save" }));
    expect(onPress).not.toHaveBeenCalled();
  });
});

describe("MessageItem", () => {
  it("shows delivered and read receipts for my messages", () => {
    const { rerender } = render(<MessageItem message={base} mine showTime startsGroup />);
    expect(screen.getByText("✓")).toBeTruthy();
    expect(screen.getByText("20:05")).toBeTruthy();
    rerender(<MessageItem message={{ ...base, readAt: "2026-01-01T20:10:00Z" }} mine showTime startsGroup />);
    expect(screen.getByText("✓✓")).toBeTruthy();
  });

  it("offers retry for failed messages", () => {
    const onRetry = jest.fn();
    const failed: ChatMessage = { ...base, status: "failed" };
    render(<MessageItem message={failed} mine onRetry={onRetry} showTime startsGroup />);
    fireEvent.press(screen.getByRole("button", { name: /Retry/ }));
    expect(onRetry).toHaveBeenCalledWith(failed);
  });

  it("hides receipts on incoming messages", () => {
    render(<MessageItem message={{ ...base, senderId: "them" }} mine={false} showTime startsGroup />);
    expect(screen.queryByText("✓")).toBeNull();
  });
});

describe("InterestPicker", () => {
  const interests: Interest[] = Array.from({ length: 12 }, (_, index) => ({
    id: index + 1,
    slug: `i${index + 1}`,
    label: `Interest ${index + 1}`
  }));

  it("toggles selections", () => {
    const onChange = jest.fn();
    render(<InterestPicker interests={interests} onChange={onChange} selectedIds={[1]} />);
    fireEvent.press(screen.getByRole("button", { name: "Interest 2" }));
    expect(onChange).toHaveBeenCalledWith([1, 2]);
    fireEvent.press(screen.getByRole("button", { name: "Interest 1" }));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("stops at 10 selections", () => {
    const onChange = jest.fn();
    render(<InterestPicker interests={interests} onChange={onChange} selectedIds={[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]} />);
    expect(screen.getByText("10 of 10 selected")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "Interest 11" }));
    expect(onChange).not.toHaveBeenCalled();
  });
});
