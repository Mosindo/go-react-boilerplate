import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import { ApiError } from "../api/client";
import { navStub, renderWithProviders } from "./testUtils";

const mockSignIn = jest.fn();
const mockSignUp = jest.fn();
jest.mock("../auth/AuthProvider", () => ({
  useAuth: () => ({
    signIn: mockSignIn,
    signUp: mockSignUp,
    signOut: jest.fn(),
    status: "signedOut",
    user: null,
    onAccountDeleted: jest.fn()
  })
}));

type LoginProps = React.ComponentProps<typeof LoginScreen>;
type RegisterProps = React.ComponentProps<typeof RegisterScreen>;
const loginProps = () =>
  ({ navigation: navStub(), route: { key: "k", name: "Login" } }) as unknown as LoginProps;
const registerProps = () =>
  ({ navigation: navStub(), route: { key: "k", name: "Register" } }) as unknown as RegisterProps;

beforeEach(() => {
  mockSignIn.mockReset();
  mockSignUp.mockReset();
});

describe("LoginScreen", () => {
  it("asks for both fields before calling the API", async () => {
    await renderWithProviders(<LoginScreen {...loginProps()} />);
    await fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(await screen.findByTestId("auth-error")).toHaveTextContent(/Saisissez votre e-mail/);
    expect(mockSignIn).not.toHaveBeenCalled();
  });

  it("signs in with the typed credentials", async () => {
    mockSignIn.mockResolvedValue(undefined);
    await renderWithProviders(<LoginScreen {...loginProps()} />);
    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "ana@example.com");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "Password123");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalledWith("ana@example.com", "Password123"));
  });

  it("shows a friendly message for wrong credentials", async () => {
    mockSignIn.mockRejectedValue(new ApiError(401, "invalid_credentials", "invalid credentials"));
    await renderWithProviders(<LoginScreen {...loginProps()} />);
    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "ana@example.com");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "wrong-password");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(await screen.findByText("E-mail ou mot de passe incorrect.")).toBeTruthy();
  });
});

describe("RegisterScreen", () => {
  it("requires the adult confirmation", async () => {
    await renderWithProviders(<RegisterScreen {...registerProps()} />);
    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "ana@example.com");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "Password123");
    await fireEvent.changeText(screen.getByTestId("auth-confirm-input"), "Password123");
    await fireEvent.press(screen.getByTestId("auth-submit-button"));
    expect(await screen.findByTestId("auth-error")).toHaveTextContent(/18 ans/);
    expect(mockSignUp).not.toHaveBeenCalled();
  });

  it("creates the account once everything is valid", async () => {
    mockSignUp.mockResolvedValue(undefined);
    await renderWithProviders(<RegisterScreen {...registerProps()} />);
    await fireEvent.changeText(screen.getByTestId("auth-email-input"), "ana@example.com");
    await fireEvent.changeText(screen.getByTestId("auth-password-input"), "Password123");
    await fireEvent.changeText(screen.getByTestId("auth-confirm-input"), "Password123");
    await fireEvent.press(screen.getByTestId("auth-adult-toggle"));
    await fireEvent.press(screen.getByTestId("auth-submit-button"));
    await waitFor(() => expect(mockSignUp).toHaveBeenCalledWith("ana@example.com", "Password123"));
  });
});
