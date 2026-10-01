import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import type { PublicProfile } from "../../api/types";
import { ThemeProvider } from "../../shared/ui";
import HomeScreen from "../HomeScreen";

const mockNavigate = jest.fn();
const mockNext = jest.fn();
const mockSwipe = jest.fn();

jest.mock("@react-navigation/native", () => ({
  ...jest.requireActual("@react-navigation/native"),
  useNavigation: () => ({ navigate: mockNavigate })
}));
jest.mock("../../hooks/useAuth", () => ({ useAuth: () => ({ accessToken: "token" }) }));
jest.mock("../../hooks/useDeviceLocation", () => ({ useDeviceLocation: () => ({ state: "idle", message: null, share: jest.fn() }) }));
jest.mock("../../shared/feedback/store", () => ({
  ...jest.requireActual("../../shared/feedback/store"),
  showToast: jest.fn()
}));
jest.mock("../../api/platform", () => ({
  discoverApi: { next: (...a: unknown[]) => mockNext(...a), swipe: (...a: unknown[]) => mockSwipe(...a) },
  profileApi: { getOwn: async () => ({ hasLocation: true }) }
}));

const profile = (id: string, name: string): PublicProfile => ({
  id,
  firstName: name,
  age: 28,
  gender: "woman",
  bio: "",
  city: "Lyon",
  distanceKm: 3,
  interests: [],
  photos: [{ id: `p-${id}`, position: 0, url: `/photos/p-${id}/file` }]
});

function renderScreen() {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false } } });
  return render(
    <ThemeProvider forceScheme="light">
      <QueryClientProvider client={client}>
        <HomeScreen />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

beforeEach(() => {
  mockNavigate.mockReset();
  mockNext.mockReset();
  mockSwipe.mockReset();
  jest.useFakeTimers();
});
afterEach(() => jest.useRealTimers());

async function flush() {
  await act(async () => {
    jest.advanceTimersByTime(500);
  });
}

describe("HomeScreen (discovery)", () => {
  it("loads candidates, shows the first one and records a like", async () => {
    mockNext.mockResolvedValueOnce([profile("a", "Camille"), profile("b", "Léa")]).mockResolvedValue([]);
    mockSwipe.mockResolvedValue({ action: "like", matched: false, alreadySwiped: false });
    renderScreen();

    expect(await screen.findByText("Camille, 28")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "J'aime" }));
    await flush();

    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("a", "like"));
    expect(await screen.findByText("Léa, 28")).toBeTruthy();
  });

  it("celebrates a match and opens the conversation on request", async () => {
    mockNext.mockResolvedValueOnce([profile("a", "Camille")]).mockResolvedValue([]);
    const match = { id: "m1", createdAt: "2026-06-15T10:00:00Z", conversationId: "c1", hasMessages: false, user: profile("a", "Camille") };
    mockSwipe.mockResolvedValue({ action: "like", matched: true, alreadySwiped: false, match });
    renderScreen();

    await screen.findByText("Camille, 28");
    fireEvent.press(screen.getByRole("button", { name: "J'aime" }));
    await flush();

    expect(await screen.findByText("C'est un match !")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "Écrire à Camille" }));
    expect(mockNavigate).toHaveBeenCalledWith("Conversation", { conversationId: "c1", user: match.user });
  });

  it("does not hammer the API when the server keeps returning the same few profiles", async () => {
    mockNext.mockResolvedValue([profile("a", "Camille"), profile("b", "Léa")]);
    renderScreen();
    await screen.findByText("Camille, 28");
    await flush();
    await flush();
    expect(mockNext.mock.calls.length).toBeLessThanOrEqual(2);
  });

  it("restores the card when a swipe cannot be saved", async () => {
    mockNext.mockResolvedValueOnce([profile("a", "Camille")]).mockResolvedValue([]);
    mockSwipe.mockRejectedValue(new Error("network down"));
    renderScreen();

    await screen.findByText("Camille, 28");
    fireEvent.press(screen.getByRole("button", { name: "Passer" }));
    await flush();

    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("a", "pass"));
    expect(await screen.findByText("Camille, 28")).toBeTruthy();
  });

  it("shows an empty state with a way to refresh when nobody is left", async () => {
    mockNext.mockResolvedValue([]);
    renderScreen();
    expect(await screen.findByText("C'est tout pour le moment")).toBeTruthy();
    mockNext.mockResolvedValueOnce([profile("z", "Nouvelle")]);
    fireEvent.press(screen.getByRole("button", { name: "Actualiser" }));
    expect(await screen.findByText("Nouvelle, 28")).toBeTruthy();
  });
});
