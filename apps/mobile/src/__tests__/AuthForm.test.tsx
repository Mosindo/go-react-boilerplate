import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { AuthForm } from "../features/auth/AuthForm";

function setup(mode: "login" | "register", onSubmit = jest.fn(async () => undefined)) {
  render(<AuthForm mode={mode} onSubmit={onSubmit} onSwitchMode={jest.fn()} onForgotPassword={jest.fn()} />);
  return onSubmit;
}

describe("AuthForm", () => {
  it("blocks submission with invalid input", async () => {
    const onSubmit = setup("register");
    fireEvent.changeText(screen.getByTestId("auth-email-input"), "not-an-email");
    fireEvent.changeText(screen.getByTestId("auth-password-input"), "short");
    fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(await screen.findByText("Enter a valid email address.")).toBeTruthy();
    expect(screen.getByText("Use at least 8 characters.")).toBeTruthy();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits normalized credentials", async () => {
    const onSubmit = setup("register");
    fireEvent.changeText(screen.getByTestId("auth-email-input"), "  Ana@Example.com ");
    fireEvent.changeText(screen.getByTestId("auth-password-input"), "correct horse");
    fireEvent.press(screen.getByTestId("auth-submit-button"));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith("ana@example.com", "correct horse"));
  });

  it("shows server errors", async () => {
    const onSubmit = jest.fn(async () => {
      throw new Error("invalid credentials");
    });
    setup("login", onSubmit);
    fireEvent.changeText(screen.getByTestId("auth-email-input"), "a@b.co");
    fireEvent.changeText(screen.getByTestId("auth-password-input"), "whatever1");
    fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(await screen.findByText("invalid credentials")).toBeTruthy();
  });
});
