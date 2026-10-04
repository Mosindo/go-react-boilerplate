import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import React from "react";
import { ApiError } from "../../../api/client";
import { LoginScreen } from "../LoginScreen";

const mockLogin = jest.fn();
jest.mock("../../../hooks/useAuth", () => ({ useAuth: () => ({ login: mockLogin }) }));

function renderScreen() {
  const navigation = { navigate: jest.fn() };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- minimal navigation stub for a screen test
  render(<LoginScreen navigation={navigation as any} route={{ key: "k", name: "Login" }} />);
  return navigation;
}

describe("LoginScreen", () => {
  beforeEach(() => mockLogin.mockReset());

  it("validates before calling the API", () => {
    renderScreen();
    fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getByText("Enter your email.")).toBeTruthy();
    expect(screen.getByText("Enter your password.")).toBeTruthy();
    expect(mockLogin).not.toHaveBeenCalled();

    fireEvent.changeText(screen.getByLabelText("Email"), "not-an-email");
    fireEvent.changeText(screen.getByLabelText("Password"), "x");
    fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    expect(screen.getByText("Enter a valid email address.")).toBeTruthy();
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it("normalizes the email and submits", async () => {
    mockLogin.mockResolvedValue(undefined);
    renderScreen();
    fireEvent.changeText(screen.getByLabelText("Email"), "  Ana@Example.COM ");
    fireEvent.changeText(screen.getByLabelText("Password"), "correct horse");
    fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() => expect(mockLogin).toHaveBeenCalledWith("ana@example.com", "correct horse"));
  });

  it("shows the server error and lets the user retry", async () => {
    mockLogin.mockRejectedValue(new ApiError(401, "unauthorized", "Invalid email or password"));
    renderScreen();
    fireEvent.changeText(screen.getByLabelText("Email"), "ana@example.com");
    fireEvent.changeText(screen.getByLabelText("Password"), "wrongpass");
    fireEvent.press(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Invalid email or password")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Sign in" }).props.accessibilityState.disabled).toBe(false);
  });

  it("links to registration and password reset", () => {
    const navigation = renderScreen();
    fireEvent.press(screen.getByRole("button", { name: "Create an account" }));
    expect(navigation.navigate).toHaveBeenCalledWith("Register");
    fireEvent.press(screen.getByRole("button", { name: "Forgot password?" }));
    expect(navigation.navigate).toHaveBeenCalledWith("ForgotPassword");
  });
});
