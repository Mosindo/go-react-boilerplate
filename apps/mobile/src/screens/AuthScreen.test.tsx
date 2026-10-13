import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { requestPasswordReset, resetPassword } from "../api/auth";
import { ApiError } from "../api/client";
import { renderWithProviders } from "../test-utils";
import AuthScreen from "./AuthScreen";

const mockSignIn = jest.fn();
const mockSignUp = jest.fn();

jest.mock("../hooks/useAuth", () => ({ useAuth: () => ({ signIn: mockSignIn, signUp: mockSignUp }) }));
jest.mock("../api/auth", () => ({
  requestPasswordReset: jest.fn().mockResolvedValue(undefined),
  resetPassword: jest.fn().mockResolvedValue(undefined)
}));

describe("AuthScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("validates before calling the API", () => {
    renderWithProviders(<AuthScreen />);
    fireEvent.press(screen.getByTestId("auth-submit"));
    expect(screen.getByText("Saisissez votre adresse e-mail.")).toBeTruthy();
    expect(screen.getByText("Saisissez votre mot de passe.")).toBeTruthy();
    expect(mockSignIn).not.toHaveBeenCalled();
  });

  it("signs in with a normalized email", async () => {
    mockSignIn.mockResolvedValue(undefined);
    renderWithProviders(<AuthScreen />);
    fireEvent.changeText(screen.getByTestId("auth-email"), "  Ana@Example.com ");
    fireEvent.changeText(screen.getByTestId("auth-password"), "Password123");
    fireEvent.press(screen.getByTestId("auth-submit"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalledWith("ana@example.com", "Password123"));
  });

  it("explains wrong credentials without leaking details", async () => {
    mockSignIn.mockRejectedValue(new ApiError(401, "invalid credentials"));
    renderWithProviders(<AuthScreen />);
    fireEvent.changeText(screen.getByTestId("auth-email"), "ana@example.com");
    fireEvent.changeText(screen.getByTestId("auth-password"), "Wrong12345");
    fireEvent.press(screen.getByTestId("auth-submit"));
    expect(await screen.findByText("E-mail ou mot de passe incorrect.")).toBeTruthy();
  });

  it("registers only with a strong password and states the age requirement", async () => {
    mockSignUp.mockResolvedValue(undefined);
    renderWithProviders(<AuthScreen />);
    fireEvent.press(screen.getByTestId("auth-switch-register"));
    expect(screen.getByText("En créant un compte, vous confirmez avoir au moins 18 ans.")).toBeTruthy();

    fireEvent.changeText(screen.getByTestId("auth-email"), "new@example.com");
    fireEvent.changeText(screen.getByTestId("auth-password"), "lettersonly");
    fireEvent.press(screen.getByTestId("auth-submit"));
    expect(screen.getByText("Utilisez au moins une lettre et un chiffre.")).toBeTruthy();
    expect(mockSignUp).not.toHaveBeenCalled();

    fireEvent.changeText(screen.getByTestId("auth-password"), "Password123");
    fireEvent.press(screen.getByTestId("auth-submit"));
    await waitFor(() => expect(mockSignUp).toHaveBeenCalledWith("new@example.com", "Password123"));
  });

  it("walks through account recovery: request a code, then choose a new password", async () => {
    renderWithProviders(<AuthScreen />);
    fireEvent.press(screen.getByText("Mot de passe oublié ?"));
    fireEvent.changeText(screen.getByTestId("auth-email"), "Ana@Example.com");
    fireEvent.press(screen.getByTestId("auth-submit"));
    await waitFor(() => expect(requestPasswordReset).toHaveBeenCalledWith("ana@example.com"));
    expect(await screen.findByRole("header", { name: "Nouveau mot de passe" })).toBeTruthy();

    fireEvent.changeText(screen.getByTestId("auth-code"), "ABCD2345");
    fireEvent.changeText(screen.getByTestId("auth-password"), "short");
    fireEvent.press(screen.getByTestId("auth-submit"));
    expect(screen.getByText("Le mot de passe doit contenir au moins 8 caractères.")).toBeTruthy();
    expect(resetPassword).not.toHaveBeenCalled();

    fireEvent.changeText(screen.getByTestId("auth-password"), "NewPassword9");
    fireEvent.press(screen.getByTestId("auth-submit"));
    await waitFor(() => expect(resetPassword).toHaveBeenCalledWith("ana@example.com", "ABCD2345", "NewPassword9"));
    expect(await screen.findByText("Mot de passe modifié. Vous pouvez vous connecter.")).toBeTruthy();
  });
});
