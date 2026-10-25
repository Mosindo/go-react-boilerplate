import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { AuthForm } from "./AuthForm";

describe("AuthForm (login)", () => {
  it("blocks submission and explains why when fields are invalid", async () => {
    const onSubmit = jest.fn();
    await render(<AuthForm loading={false} mode="login" onSubmit={onSubmit} />);

    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "not-an-email");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("Cet email ne semble pas valide.")).toBeTruthy();
    expect(screen.getByText("Saisissez votre mot de passe.")).toBeTruthy();
  });

  it("submits a trimmed, lower-cased email with the password", async () => {
    const onSubmit = jest.fn();
    await render(<AuthForm loading={false} mode="login" onSubmit={onSubmit} />);

    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "  Jane@Example.com ");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "secret-pass");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));

    expect(onSubmit).toHaveBeenCalledWith({ email: "jane@example.com", password: "secret-pass" });
  });

  it("shows the server error", async () => {
    await render(
      <AuthForm loading={false} mode="login" onSubmit={jest.fn()} submitError="Email ou mot de passe incorrect." />
    );
    expect(screen.getByTestId("auth-error")).toHaveTextContent("Email ou mot de passe incorrect.");
  });
});

describe("AuthForm (register)", () => {
  it("requires matching passwords of at least 8 characters", async () => {
    const onSubmit = jest.fn();
    await render(<AuthForm loading={false} mode="register" onSubmit={onSubmit} />);

    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "jane@example.com");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "short");
    await fireEvent.changeText(screen.getByTestId("auth-confirm-input"), "different");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("Au moins 8 caractères.")).toBeTruthy();
    expect(screen.getByText("Les mots de passe ne correspondent pas.")).toBeTruthy();
  });
});
