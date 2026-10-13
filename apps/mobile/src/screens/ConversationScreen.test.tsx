import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { fetchMessages, markConversationRead, sendMessage } from "../api/platform";
import type { ChatMessage } from "../api/types";
import { renderWithProviders } from "../test-utils";
import ConversationScreen from "./ConversationScreen";

jest.mock("@react-navigation/native", () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn(), popToTop: jest.fn() }),
  useRoute: () => ({ params: { conversationId: "c1", userId: "u2", name: "Bob", photoId: null } }),
  useFocusEffect: jest.fn()
}));
jest.mock("../api/platform", () => ({
  fetchMessages: jest.fn(),
  sendMessage: jest.fn(),
  markConversationRead: jest.fn().mockResolvedValue(undefined),
  hideConversation: jest.fn(),
  unmatch: jest.fn()
}));
jest.mock("../hooks/useAuth", () => ({ useAuth: () => ({ user: { id: "me" } }) }));
jest.mock("../hooks/queries", () => ({ useMatches: () => ({ data: [] }), useConversations: () => ({ data: [] }) }));
jest.mock("../features/safety/SafetyMenu", () => ({ SafetyMenu: () => null }));

const msg = (id: number, senderId: string, body: string, read = false): ChatMessage => ({
  id,
  conversationId: "c1",
  senderId,
  body,
  createdAt: new Date(2026, 0, 1, 10, id).toISOString(),
  read
});

describe("ConversationScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("renders the history with read receipts and marks the conversation read", async () => {
    jest.mocked(fetchMessages).mockResolvedValue({ messages: [msg(1, "u2", "Salut !"), msg(2, "me", "Coucou", true)], nextBefore: null });
    renderWithProviders(<ConversationScreen />);
    expect(await screen.findByText("Salut !")).toBeTruthy();
    expect(screen.getByText("Coucou")).toBeTruthy();
    expect(screen.getByText(/Lu$/)).toBeTruthy();
    await waitFor(() => expect(markConversationRead).toHaveBeenCalledWith("c1"));
  });

  it("sends a trimmed message optimistically", async () => {
    jest.mocked(fetchMessages).mockResolvedValue({ messages: [], nextBefore: null });
    jest.mocked(sendMessage).mockResolvedValue(msg(3, "me", "Ça va ?"));
    renderWithProviders(<ConversationScreen />);
    await screen.findByTestId("chat-input");

    fireEvent.changeText(screen.getByTestId("chat-input"), "  Ça va ?  ");
    fireEvent.press(screen.getByTestId("chat-send"));
    await waitFor(() => expect(sendMessage).toHaveBeenCalledWith("c1", "Ça va ?"));
    expect(await screen.findByText("Ça va ?")).toBeTruthy();
  });

  it("keeps the draft and shows the error when sending fails", async () => {
    jest.mocked(fetchMessages).mockResolvedValue({ messages: [], nextBefore: null });
    jest.mocked(sendMessage).mockRejectedValue(new Error("Connexion impossible"));
    renderWithProviders(<ConversationScreen />);
    await screen.findByTestId("chat-input");

    fireEvent.changeText(screen.getByTestId("chat-input"), "Bonjour");
    fireEvent.press(screen.getByTestId("chat-send"));
    expect(await screen.findByText("Connexion impossible")).toBeTruthy();
    expect(screen.getByTestId("chat-input").props.value).toBe("Bonjour");
  });
});
