import { fireEvent, render, screen } from "@testing-library/react-native";
import React from "react";
import { Composer } from "../Composer";

describe("Composer", () => {
  it("disables send until there is text, trims, sends and clears", () => {
    const onSend = jest.fn();
    render(<Composer onSend={onSend} />);
    const send = screen.getByRole("button", { name: "Send message" });
    expect(send.props.accessibilityState.disabled).toBe(true);
    fireEvent.press(send);
    expect(onSend).not.toHaveBeenCalled();

    const input = screen.getByLabelText("Message");
    fireEvent.changeText(input, "   ");
    expect(screen.getByRole("button", { name: "Send message" }).props.accessibilityState.disabled).toBe(true);

    fireEvent.changeText(input, "  hello there  ");
    fireEvent.press(screen.getByRole("button", { name: "Send message" }));
    expect(onSend).toHaveBeenCalledWith("hello there");
    expect(screen.getByLabelText("Message").props.value).toBe("");
  });

  it("limits messages to 2000 characters", () => {
    render(<Composer onSend={jest.fn()} />);
    expect(screen.getByLabelText("Message").props.maxLength).toBe(2000);
  });
});
