import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { ApiError } from "../../api/client";
import type { Message, PublicProfile } from "../../api/types";
import { ThemeProvider } from "../../shared/ui";
import ConversationScreen from "../ConversationScreen";

const mockMessages = jest.fn();
const mockSend = jest.fn();
const mockMarkRead = jest.fn();

jest.mock("@react-navigation/native", () => {
  const react = jest.requireActual<typeof import("react")>("react");
  return { useFocusEffect: (callback: () => void | (() => void)) => react.useEffect(callback, [callback]) };
});
jest.mock("../../hooks/useAuth", () => ({ useAuth: () => ({ user: { id: "me" }, accessToken: "token" }) }));
jest.mock("../../shared/feedback/store", () => ({ showToast: jest.fn(), showGlobalError: jest.fn() }));
jest.mock("../../api/platform", () => ({
  chatApi: {
    messages: (...a: unknown[]) => mockMessages(...a),
    send: (...a: unknown[]) => mockSend(...a),
    markRead: (...a: unknown[]) => mockMarkRead(...a),
    clear: jest.fn()
  },
  discoverApi: { matches: jest.fn(), unmatch: jest.fn() },
  profileApi: { getPublic: jest.fn() },
  safetyApi: { block: jest.fn(), report: jest.fn() }
}));

const partner: PublicProfile = {
  id: "them",
  firstName: "Camille",
  age: 29,
  gender: "woman",
  bio: "",
  city: "",
  distanceKm: null,
  interests: [],
  photos: []
};

const message = (id: string, senderId: string, body: string, readAt: string | null = null): Message => ({
  id,
  conversationId: "c1",
  senderId,
  body,
  createdAt: "2026-06-15T10:00:00Z",
  readAt
});

const navigation = { setOptions: jest.fn(), goBack: jest.fn(), navigate: jest.fn() };

function renderScreen() {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false }, mutations: { gcTime: Infinity } } });
  const props = { navigation, route: { key: "k", name: "Conversation", params: { conversationId: "c1", user: partner } } };
  return render(
    <ThemeProvider forceScheme="light">
      <QueryClientProvider client={client}>
        <ConversationScreen {...(props as unknown as React.ComponentProps<typeof ConversationScreen>)} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

beforeEach(() => {
  mockMessages.mockReset();
  mockSend.mockReset();
  mockMarkRead.mockReset().mockResolvedValue({ marked: 1 });
  navigation.goBack.mockReset();
});

describe("ConversationScreen", () => {
  it("renders history with read status on my messages", async () => {
    mockMessages.mockResolvedValue({
      messages: [message("1", "them", "Salut !"), message("2", "me", "Coucou", "2026-06-15T10:05:00Z")],
      hasMore: false
    });
    renderScreen();
    expect(await screen.findByText("Salut !")).toBeTruthy();
    expect(screen.getByText("Coucou")).toBeTruthy();
    expect(screen.getByText(/Lu$/)).toBeTruthy();
  });

  it("marks incoming unread messages as read", async () => {
    mockMessages.mockResolvedValue({ messages: [message("1", "them", "Salut !")], hasMore: false });
    renderScreen();
    await screen.findByText("Salut !");
    await waitFor(() => expect(mockMarkRead).toHaveBeenCalledWith("c1"));
  });

  it("sends a trimmed message and shows it once confirmed", async () => {
    mockMessages.mockResolvedValue({ messages: [], hasMore: false });
    mockSend.mockResolvedValue(message("9", "me", "Bonjour"));
    renderScreen();

    await screen.findByText(/Dites bonjour/);
    fireEvent.changeText(screen.getByLabelText("Votre message"), "  Bonjour  ");
    fireEvent.press(screen.getByRole("button", { name: "Envoyer" }));

    await waitFor(() => expect(mockSend).toHaveBeenCalledWith("c1", "Bonjour"));
    expect(await screen.findByText(/Bonjour/)).toBeTruthy();
  });

  it("does not send blank messages", async () => {
    mockMessages.mockResolvedValue({ messages: [], hasMore: false });
    renderScreen();
    await screen.findByText(/Dites bonjour/);
    fireEvent.changeText(screen.getByLabelText("Votre message"), "   ");
    expect(screen.getByRole("button", { name: "Envoyer" }).props.accessibilityState).toMatchObject({ disabled: true });
    expect(mockSend).not.toHaveBeenCalled();
  });

  it("keeps a failed message visible and retryable", async () => {
    mockMessages.mockResolvedValue({ messages: [], hasMore: false });
    mockSend.mockRejectedValueOnce(new ApiError(0, "offline")).mockResolvedValueOnce(message("9", "me", "Réessai"));
    renderScreen();

    await screen.findByText(/Dites bonjour/);
    fireEvent.changeText(screen.getByLabelText("Votre message"), "Réessai");
    fireEvent.press(screen.getByRole("button", { name: "Envoyer" }));
    expect(await screen.findByText(/Échec de l'envoi/)).toBeTruthy();

    fireEvent.press(screen.getByText("Réessai"));
    await waitFor(() => expect(mockSend).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByText(/Échec de l'envoi/)).toBeNull());
  });

  it("explains when the match has ended instead of showing a broken chat", async () => {
    mockMessages.mockRejectedValue(new ApiError(404, "conversation not found"));
    renderScreen();
    expect(await screen.findByText("Conversation fermée")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "Retour" }));
    expect(navigation.goBack).toHaveBeenCalled();
  });
});
