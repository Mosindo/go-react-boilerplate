import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FeedbackProvider } from "../components/Feedback";
import { LoginScreen, RegisterScreen } from "../screens/AuthScreens";

const mockSignIn = jest.fn(async (_email: string, _password: string) => undefined);
const mockSignUp = jest.fn(async (_email: string, _password: string) => undefined);

jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));
jest.mock("../auth/AuthProvider", () => ({ useAuth: () => ({ signIn: mockSignIn, signUp: mockSignUp }) }));

const navigation = { navigate: jest.fn() };
const wrap = (ui: React.ReactElement) => (
  <QueryClientProvider client={new QueryClient()}>
    <FeedbackProvider>{ui}</FeedbackProvider>
  </QueryClientProvider>
);

beforeEach(() => jest.clearAllMocks());

describe("auth screens", () => {
  it("login validates before calling the API", () => {
    render(wrap(<LoginScreen navigation={navigation as never} route={{ key: "k", name: "Login" }} />));
    fireEvent.press(screen.getByTestId("login-submit"));
    expect(screen.getByText("Entrez votre email.")).toBeTruthy();
    expect(mockSignIn).not.toHaveBeenCalled();
  });

  it("login normalises the email", async () => {
    render(wrap(<LoginScreen navigation={navigation as never} route={{ key: "k", name: "Login" }} />));
    fireEvent.changeText(screen.getByTestId("login-email"), "  Léa@Example.com ");
    fireEvent.changeText(screen.getByTestId("login-password"), "secret-password");
    fireEvent.press(screen.getByTestId("login-submit"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalledWith("léa@example.com", "secret-password"));
  });

  it("register requires the adult confirmation and a valid password", () => {
    render(wrap(<RegisterScreen navigation={navigation as never} route={{ key: "k", name: "Register" }} />));
    fireEvent.changeText(screen.getByTestId("register-email"), "a@b.co");
    fireEvent.changeText(screen.getByTestId("register-password"), "short");
    fireEvent.press(screen.getByTestId("register-submit"));
    expect(screen.getByText("Le mot de passe doit contenir au moins 8 caractères.")).toBeTruthy();
    expect(screen.getByText("Vous devez confirmer avoir 18 ans ou plus.")).toBeTruthy();
    expect(mockSignUp).not.toHaveBeenCalled();
  });

  it("register sends a normalised email once everything is valid", async () => {
    render(wrap(<RegisterScreen navigation={navigation as never} route={{ key: "k", name: "Register" }} />));
    fireEvent.changeText(screen.getByTestId("register-email"), " New@User.io ");
    fireEvent.changeText(screen.getByTestId("register-password"), "long-enough-pass");
    fireEvent.press(screen.getByTestId("register-adult"));
    fireEvent.press(screen.getByTestId("register-submit"));
    await waitFor(() => expect(mockSignUp).toHaveBeenCalledWith("new@user.io", "long-enough-pass"));
  });
});
