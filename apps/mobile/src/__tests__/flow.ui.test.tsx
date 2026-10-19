import React from "react";
import { act, fireEvent, screen, waitFor } from "@testing-library/react-native";
import ChatScreen from "../screens/main/ChatScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import type { Conversation, Match, MessagesPage, PublicProfile } from "../api/types";
import { navStub, renderWithProviders, screenProps } from "./testUtils";

type DiscoverProps = React.ComponentProps<typeof DiscoverScreen>;
type ChatProps = React.ComponentProps<typeof ChatScreen>;

const mockDiscover = jest.fn();
const mockSwipe = jest.fn();
const mockListMessages = jest.fn();
const mockGetConversation = jest.fn();
const mockSendMessage = jest.fn();
const mockMarkRead = jest.fn();

jest.mock("../api/social", () => ({
  discover: (...args: unknown[]) => mockDiscover(...args),
  swipe: (...args: unknown[]) => mockSwipe(...args),
  listMessages: (...args: unknown[]) => mockListMessages(...args),
  getConversation: (...args: unknown[]) => mockGetConversation(...args),
  sendMessage: (...args: unknown[]) => mockSendMessage(...args),
  markRead: (...args: unknown[]) => mockMarkRead(...args),
  listConversations: jest.fn().mockResolvedValue({ conversations: [] }),
  blockUser: jest.fn()
}));
jest.mock("../auth/AuthProvider", () => ({
  useAuth: () => ({ user: { id: "me", email: "me@test.invalid", createdAt: "" }, signOut: jest.fn() })
}));

const profile = (id: string, name: string): PublicProfile => ({
  userId: id,
  firstName: name,
  age: 29,
  bio: "",
  city: "Lyon",
  distanceKm: 10,
  interests: [{ id: 1, slug: "travel", label: "Voyages" }],
  photos: [{ id: `photo-${id}`, position: 0 }]
});

beforeEach(() => {
  jest.clearAllMocks();
});

describe("Discover → like → match", () => {
  it("shows the next profile, sends the like and celebrates a match", async () => {
    mockDiscover
      .mockResolvedValueOnce([profile("u1", "Camille"), profile("u2", "Inès")])
      .mockResolvedValue([]);
    const match: Match = { id: "m1", conversationId: "c1", createdAt: "", profile: profile("u1", "Camille") };
    mockSwipe.mockResolvedValue({ matched: true, duplicate: false, match });
    const navigation = navStub();

    await renderWithProviders(
      <DiscoverScreen {...screenProps<DiscoverProps>({ name: "Discover" }, navigation)} />
    );

    expect(await screen.findByText("Camille, 29")).toBeTruthy();
    await fireEvent.press(screen.getByTestId("like-button"));

    // The card flies out (Animated.timing), then the like is sent.
    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("u1", "like"), { timeout: 2000 });
    expect(await screen.findByTestId("match-modal")).toBeTruthy();
    expect(screen.getByText(/C'est un match/)).toBeTruthy();

    await fireEvent.press(screen.getByTestId("match-message"));
    expect(navigation.navigate).toHaveBeenCalledWith("Chat", { conversationId: "c1", name: "Camille" });
  });

  it("passes without opening a match modal", async () => {
    mockDiscover.mockResolvedValueOnce([profile("u1", "Camille")]).mockResolvedValue([]);
    mockSwipe.mockResolvedValue({ matched: false, duplicate: false });
    await renderWithProviders(<DiscoverScreen {...screenProps<DiscoverProps>({ name: "Discover" })} />);
    await screen.findByText("Camille, 29");
    await fireEvent.press(screen.getByTestId("pass-button"));
    await waitFor(() => expect(mockSwipe).toHaveBeenCalledWith("u1", "pass"), { timeout: 2000 });
    expect(screen.queryByTestId("match-modal")).toBeNull();
    expect(await screen.findByText("Vous avez tout vu pour le moment")).toBeTruthy();
  });

  it("explains how to continue when the profile has no photo yet", async () => {
    const { ApiError } = jest.requireActual("../api/client");
    mockDiscover.mockRejectedValue(new ApiError(409, "profile_incomplete", "incomplete"));
    await renderWithProviders(<DiscoverScreen {...screenProps<DiscoverProps>({ name: "Discover" })} />);
    expect(await screen.findByText("Ajoutez une photo")).toBeTruthy();
  });
});

describe("Chat", () => {
  const conversation: Conversation = {
    id: "c1",
    matchId: "m1",
    createdAt: "",
    updatedAt: "",
    profile: profile("u1", "Camille"),
    unreadCount: 0
  };
  const page: MessagesPage = {
    messages: [
      { id: "m-1", conversationId: "c1", senderId: "u1", body: "Salut !", createdAt: "2026-01-01T10:00:00Z" }
    ],
    peerLastReadAt: "2026-01-01T10:00:00Z"
  };

  it("lists messages, marks them read and sends a trimmed message", async () => {
    mockListMessages.mockResolvedValue(page);
    mockGetConversation.mockResolvedValue(conversation);
    mockMarkRead.mockResolvedValue(undefined);
    mockSendMessage.mockResolvedValue({
      id: "m-2",
      conversationId: "c1",
      senderId: "me",
      body: "Coucou",
      createdAt: "2026-01-01T10:01:00Z"
    });

    const navigation = navStub();
    await renderWithProviders(
      <ChatScreen
        {...screenProps<ChatProps>(
          { name: "Chat", params: { conversationId: "c1", name: "Camille" } },
          navigation
        )}
      />
    );

    expect(await screen.findByText("Salut !")).toBeTruthy();
    await waitFor(() => expect(mockMarkRead).toHaveBeenCalledWith("c1"));

    await fireEvent.changeText(screen.getByTestId("chat-input"), "  Coucou  ");
    await fireEvent.press(screen.getByTestId("chat-send"));
    await waitFor(() => expect(mockSendMessage).toHaveBeenCalledWith("c1", "Coucou"));
    expect(await screen.findByText("Coucou")).toBeTruthy();
    await act(async () => undefined);
    expect(screen.getByTestId("chat-input").props.value).toBe("");
  });

  it("does not send blank messages", async () => {
    mockListMessages.mockResolvedValue({ messages: [], peerLastReadAt: "" });
    mockGetConversation.mockResolvedValue(conversation);
    mockMarkRead.mockResolvedValue(undefined);
    await renderWithProviders(
      <ChatScreen
        {...screenProps<ChatProps>({ name: "Chat", params: { conversationId: "c1", name: "Camille" } })}
      />
    );
    await screen.findByText(/Envoyez le premier message/);
    await fireEvent.changeText(screen.getByTestId("chat-input"), "   ");
    await fireEvent.press(screen.getByTestId("chat-send"));
    expect(mockSendMessage).not.toHaveBeenCalled();
  });
});
