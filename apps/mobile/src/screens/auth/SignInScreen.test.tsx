import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import React from "react";
import * as authApi from "../../api/auth";
import type { AuthSession } from "../../api/types";
import SignInScreen from "./SignInScreen";

jest.mock("../../api/auth");
const mockStartSession = jest.fn();
jest.mock("../../state/AuthProvider", () => ({
  useAuth: () => ({ startSession: mockStartSession })
}));

const session: AuthSession = {
  accessToken: "a",
  refreshToken: "r",
  user: {
    id: "u1",
    email: "jane@example.com",
    createdAt: "2026-01-01T00:00:00Z",
    hasProfile: false,
    hasPreferences: false,
    hasPhotos: false,
    onboardingComplete: false
  }
};

const navigation = { navigate: jest.fn() } as never;
const route = { key: "SignIn", name: "SignIn" } as never;

describe("SignInScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("rejects an invalid email without calling the API", async () => {
    render(<SignInScreen navigation={navigation} route={route} />);
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "not-an-email");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "whatever123");
    fireEvent.press(screen.getByTestId("signin-submit"));
    expect(await screen.findByText("Cette adresse e-mail n'est pas valide.")).toBeTruthy();
    expect(authApi.login).not.toHaveBeenCalled();
  });

  it("signs in with a normalized email", async () => {
    jest.mocked(authApi.login).mockResolvedValue(session);
    render(<SignInScreen navigation={navigation} route={route} />);
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "  Jane@Example.com ");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "correct-horse");
    fireEvent.press(screen.getByTestId("signin-submit"));
    await waitFor(() => expect(mockStartSession).toHaveBeenCalledWith(session));
    expect(authApi.login).toHaveBeenCalledWith("jane@example.com", "correct-horse");
  });

  it("shows the server error on bad credentials", async () => {
    jest.mocked(authApi.login).mockRejectedValue(new Error("invalid email or password"));
    render(<SignInScreen navigation={navigation} route={route} />);
    fireEvent.changeText(screen.getByLabelText("Adresse e-mail"), "jane@example.com");
    fireEvent.changeText(screen.getByLabelText("Mot de passe"), "nope-nope");
    fireEvent.press(screen.getByTestId("signin-submit"));
    expect(await screen.findByText("invalid email or password")).toBeTruthy();
    expect(mockStartSession).not.toHaveBeenCalled();
  });
});
