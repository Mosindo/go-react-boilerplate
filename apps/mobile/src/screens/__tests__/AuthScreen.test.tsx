import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { ApiError } from "../../api/client";
import { ThemeProvider } from "../../shared/ui";
import AuthScreen from "../AuthScreen";

const mockSignIn = jest.fn();
const mockLogin = jest.fn();
const mockRegister = jest.fn();

jest.mock("../../hooks/useAuth", () => ({
  useAuth: () => ({ signIn: mockSignIn, sessionNotice: null, clearSessionNotice: jest.fn() })
}));
jest.mock("../../api/auth", () => ({
  login: (...args: unknown[]) => mockLogin(...args),
  register: (...args: unknown[]) => mockRegister(...args),
  requestPasswordReset: jest.fn(),
  resetPassword: jest.fn()
}));

const session = { accessToken: "a", refreshToken: "r", user: { id: "u1" } };

function renderScreen() {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity }, mutations: { gcTime: Infinity } } });
  render(
    <ThemeProvider forceScheme="light">
      <QueryClientProvider client={client}>
        <AuthScreen />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

beforeEach(() => {
  mockSignIn.mockReset();
  mockLogin.mockReset();
  mockRegister.mockReset();
});

describe("AuthScreen", () => {
  it("blocks submission with an invalid email and explains why", async () => {
    renderScreen();
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "not-an-email");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "whatever1");
    fireEvent.press(screen.getByRole("button", { name: "Se connecter" }));
    expect(await screen.findByText("Adresse e-mail invalide.")).toBeTruthy();
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it("logs in with a trimmed email and stores the session", async () => {
    mockLogin.mockResolvedValue(session);
    renderScreen();
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "  jane@example.com ");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "Password123");
    fireEvent.press(screen.getByRole("button", { name: "Se connecter" }));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalledWith(session));
    expect(mockLogin).toHaveBeenCalledWith("jane@example.com", "Password123");
  });

  it("shows the server error without crashing", async () => {
    mockLogin.mockRejectedValue(new ApiError(401, "invalid credentials"));
    renderScreen();
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "jane@example.com");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "wrong-password");
    fireEvent.press(screen.getByRole("button", { name: "Se connecter" }));
    expect(await screen.findByText("invalid credentials")).toBeTruthy();
    expect(mockSignIn).not.toHaveBeenCalled();
  });

  it("enforces the password rule when registering", async () => {
    renderScreen();
    fireEvent.press(screen.getByRole("button", { name: "Créer un compte" }));
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "jane@example.com");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "short");
    fireEvent.press(screen.getByRole("button", { name: "Créer mon compte" }));
    expect(await screen.findByText("Au moins 8 caractères.")).toBeTruthy();
    expect(mockRegister).not.toHaveBeenCalled();
    await act(async () => {});
  });

  it("registers with a valid password", async () => {
    mockRegister.mockResolvedValue(session);
    renderScreen();
    fireEvent.press(screen.getByRole("button", { name: "Créer un compte" }));
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "jane@example.com");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "Password123");
    fireEvent.press(screen.getByRole("button", { name: "Créer mon compte" }));
    await waitFor(() => expect(mockRegister).toHaveBeenCalledWith("jane@example.com", "Password123"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalled());
  });
});
