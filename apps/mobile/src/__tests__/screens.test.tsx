import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PublicProfile } from "../api/types";

const mockNavigate = jest.fn();
jest.mock("@react-navigation/native", () => ({
  useNavigation: () => ({ navigate: mockNavigate })
}));
jest.mock("react-native-safe-area-context", () => {
  const { View } = jest.requireActual("react-native");
  return { SafeAreaView: View, SafeAreaProvider: View };
});

const mockFeed = jest.fn();
const mockSwipe = jest.fn();
const mockLogin = jest.fn();
jest.mock("../api/endpoints", () => ({
  discoveryApi: {
    feed: (...a: unknown[]) => mockFeed(...a),
    swipe: (...a: unknown[]) => mockSwipe(...a),
    matches: jest.fn()
  },
  authApi: { login: (...a: unknown[]) => mockLogin(...a) },
  profileApi: { get: jest.fn().mockResolvedValue(null), interests: jest.fn().mockResolvedValue([]) }
}));

const mockSignIn = jest.fn();
jest.mock("../state/session", () => ({ useSession: () => ({ signIn: mockSignIn }) }));

import { DiscoverScreen } from "../screens/DiscoverScreen";
import { LoginScreen } from "../screens/AuthScreens";
import { ProfileForm } from "../components/ProfileForm";
import { ToastProvider } from "../ui/Toast";

const profile = (id: string, name: string): PublicProfile => ({
  userId: id,
  firstName: name,
  age: 30,
  gender: "woman",
  bio: "",
  city: "Paris",
  distanceKm: 5,
  interests: ["music"],
  photos: [{ id: `ph-${id}`, url: `/photos/ph-${id}`, position: 0, width: 400, height: 500 }]
});

async function renderWithProviders(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return await render(
    <QueryClientProvider client={client}>
      <ToastProvider>{ui}</ToastProvider>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  jest.clearAllMocks();
});

describe("DiscoverScreen", () => {
  it("shows the first profile, likes it and surfaces a match", async () => {
    mockFeed
      .mockResolvedValueOnce([profile("u1", "Bob"), profile("u2", "Carl")])
      .mockResolvedValue([]);
    mockSwipe.mockResolvedValue({ matched: true, matchId: "m1", conversationId: "c1" });

    await renderWithProviders(<DiscoverScreen />);
    expect(await screen.findByText("Bob, 30")).toBeTruthy();

    await fireEvent.press(screen.getByTestId("like-button"));
    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("u1", "like"));
    expect(await screen.findByText("It's a match!")).toBeTruthy();

    await fireEvent.press(screen.getByTestId("match-send-message"));
    expect(mockNavigate).toHaveBeenCalledWith(
      "Chat",
      expect.objectContaining({
        conversationId: "c1",
        peer: expect.objectContaining({ userId: "u1" })
      })
    );
  });

  it("passes without a match modal", async () => {
    mockFeed.mockResolvedValueOnce([profile("u1", "Bob")]).mockResolvedValue([]);
    mockSwipe.mockResolvedValue({ matched: false });
    await renderWithProviders(<DiscoverScreen />);
    await screen.findByText("Bob, 30");
    await fireEvent.press(screen.getByTestId("pass-button"));
    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("u1", "pass"));
    expect(await screen.findByText("You're all caught up")).toBeTruthy();
    expect(screen.queryByText("It's a match!")).toBeNull();
  });

  it("shows an empty state when nobody is available", async () => {
    mockFeed.mockResolvedValue([]);
    await renderWithProviders(<DiscoverScreen />);
    expect(await screen.findByText("You're all caught up")).toBeTruthy();
  });

  it("shows an error state with retry when loading fails", async () => {
    mockFeed.mockRejectedValueOnce(new Error("boom")).mockResolvedValue([profile("u1", "Bob")]);
    await renderWithProviders(<DiscoverScreen />);
    expect(await screen.findByText("boom")).toBeTruthy();
    await fireEvent.press(screen.getByText("Try again"));
    expect(await screen.findByText("Bob, 30")).toBeTruthy();
  });
});

describe("LoginScreen", () => {
  const navigation = { navigate: jest.fn() } as never;
  const route = { key: "k", name: "Login" } as never;

  it("validates before calling the API", async () => {
    await renderWithProviders(<LoginScreen navigation={navigation} route={route} />);
    await fireEvent.press(screen.getByTestId("login-submit"));
    expect(await screen.findByText("Enter your email.")).toBeTruthy();
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it("normalizes the email and signs in", async () => {
    mockLogin.mockResolvedValue({ accessToken: "a", refreshToken: "r", user: { id: "u" } });
    await renderWithProviders(<LoginScreen navigation={navigation} route={route} />);
    await fireEvent.changeText(screen.getByTestId("login-email"), "  Me@Example.COM ");
    await fireEvent.changeText(screen.getByTestId("login-password"), "Password123");
    await fireEvent.press(screen.getByTestId("login-submit"));
    await waitFor(() => expect(mockLogin).toHaveBeenCalledWith("me@example.com", "Password123"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalled());
  });

  it("shows the server error", async () => {
    mockLogin.mockRejectedValue(new Error("invalid credentials"));
    await renderWithProviders(<LoginScreen navigation={navigation} route={route} />);
    await fireEvent.changeText(screen.getByTestId("login-email"), "me@example.com");
    await fireEvent.changeText(screen.getByTestId("login-password"), "wrong");
    await fireEvent.press(screen.getByTestId("login-submit"));
    expect(await screen.findByText("invalid credentials")).toBeTruthy();
  });
});

describe("ProfileForm", () => {
  it("blocks underage users and submits a valid profile", async () => {
    const onSubmit = jest.fn();
    await renderWithProviders(
      <ProfileForm submitLabel="Continue" saving={false} onSubmit={onSubmit} />
    );

    await fireEvent.changeText(screen.getByTestId("profile-first-name"), "Zoé");
    await fireEvent.changeText(screen.getByTestId("profile-birth-date"), "20150101");
    await fireEvent.press(screen.getByText("Woman"));
    await fireEvent.press(screen.getByTestId("profile-submit"));
    expect(await screen.findByText(/at least 18/)).toBeTruthy();
    expect(onSubmit).not.toHaveBeenCalled();

    await fireEvent.changeText(screen.getByTestId("profile-birth-date"), "19950310");
    await fireEvent.press(screen.getByTestId("profile-submit"));
    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ firstName: "Zoé", birthDate: "1995-03-10", gender: "woman" })
      )
    );
  });
});
