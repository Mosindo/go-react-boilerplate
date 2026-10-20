import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  View
} from "react-native";
import {
  listChatMessages,
  listChats,
  listUsers,
  sendChatMessage,
  type ChatMessage,
  type ChatSummary,
  type PlatformUser
} from "../api/platform";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Avatar, Button, Card, Input, ListItem, MessageItem, Notice, Text, colors, radii, spacing } from "../shared/ui";

type ChatScreenProps = {
  token: string;
  currentUserId: string;
};

type ChatTarget = {
  user: PlatformUser;
  lastMessage?: ChatSummary["lastMessage"];
};

const CHAT_LIST_POLL_MS = 8000;
const CHAT_MESSAGES_POLL_MS = 3000;

function formatChatError(error: unknown, fallback: string): string {
  if (!(error instanceof Error) || !error.message) {
    return fallback;
  }
  const message = error.message.toLowerCase();
  if (message.includes("network") || message.includes("failed to fetch")) {
    return "Network error. Check connection and retry.";
  }
  if (message.includes("timeout")) {
    return "Request timed out. Please retry.";
  }
  return error.message;
}

function chatSummaryToTarget(chat: ChatSummary): ChatTarget {
  return { user: chat.user, lastMessage: chat.lastMessage };
}

export default function ChatScreen({ token, currentUserId }: ChatScreenProps) {
  const [chatList, setChatList] = useState<ChatSummary[]>([]);
  const [directory, setDirectory] = useState<PlatformUser[]>([]);
  const [selectedChat, setSelectedChat] = useState<ChatTarget | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [backgroundError, setBackgroundError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const lobbyInFlight = useRef(false);
  const messagesInFlight = useRef(false);

  const loadLobby = useCallback(async (silent = false) => {
    if (lobbyInFlight.current) {
      return;
    }

    lobbyInFlight.current = true;
    if (!silent) {
      setLoading(true);
      setError(null);
    }

    try {
      const [nextChats, nextUsers] = await Promise.all([listChats(token), listUsers(token)]);
      const filteredUsers = nextUsers.filter((user) => user.id !== currentUserId);

      setChatList(nextChats);
      setDirectory(filteredUsers);
      setBackgroundError(null);
      setSelectedChat((current) => {
        if (!current) {
          return null;
        }

        const existingChat = nextChats.find((chat) => chat.user.id === current.user.id);
        if (existingChat) {
          return chatSummaryToTarget(existingChat);
        }

        const existingUser = filteredUsers.find((user) => user.id === current.user.id);
        if (existingUser) {
          return { user: existingUser, lastMessage: current.lastMessage };
        }

        return current;
      });
    } catch (loadError) {
      const formattedError = formatChatError(loadError, "could not load conversations");
      if (silent) {
        setBackgroundError(formattedError);
      } else {
        setError(formattedError);
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
      lobbyInFlight.current = false;
    }
  }, [token, currentUserId]);

  const loadMessages = useCallback(
    async (chat: ChatTarget, silent = false): Promise<boolean> => {
      if (messagesInFlight.current) {
        return true;
      }

      messagesInFlight.current = true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }

      try {
        const nextMessages = await listChatMessages(token, chat.user.id);
        setMessages(nextMessages);
        setBackgroundError(null);
        return true;
      } catch (loadError) {
        const formattedError = formatChatError(loadError, "could not load messages");
        if (silent) {
          setBackgroundError(formattedError);
        } else {
          setError(formattedError);
        }
        return false;
      } finally {
        if (!silent) {
          setLoading(false);
        }
        messagesInFlight.current = false;
      }
    },
    [token]
  );

  useEffect(() => {
    loadLobby();
  }, [loadLobby]);

  async function openChat(target: ChatTarget) {
    setSelectedChat(target);
    setError(null);
    setBackgroundError(null);
    const ok = await loadMessages(target);
    if (!ok) {
      setMessages([]);
    }
  }

  async function onSend() {
    if (!selectedChat) {
      return;
    }

    const content = draft.trim();
    if (!content) {
      return;
    }

    setSending(true);
    setError(null);
    setBackgroundError(null);
    try {
      const sent = await sendChatMessage(token, selectedChat.user.id, content);
      setMessages((prev) => [...prev, sent]);
      setDraft("");
      await loadLobby(true);
      await loadMessages(selectedChat, true);
    } catch (sendError) {
      setError(formatChatError(sendError, "could not send message"));
    } finally {
      setSending(false);
    }
  }

  async function reloadCurrentView() {
    setBackgroundError(null);
    if (selectedChat) {
      await loadMessages(selectedChat);
      return;
    }
    await loadLobby();
  }

  useEffect(() => {
    if (selectedChat) {
      return;
    }

    const intervalId = setInterval(() => {
      void loadLobby(true);
    }, CHAT_LIST_POLL_MS);

    return () => {
      clearInterval(intervalId);
    };
  }, [selectedChat, loadLobby]);

  useEffect(() => {
    if (!selectedChat) {
      return;
    }

    const intervalId = setInterval(() => {
      void loadMessages(selectedChat, true);
      void loadLobby(true);
    }, CHAT_MESSAGES_POLL_MS);

    return () => {
      clearInterval(intervalId);
    };
  }, [selectedChat, loadMessages, loadLobby]);

  const availableContacts = useMemo(
    () => directory.filter((user) => !chatList.some((chat) => chat.user.id === user.id)),
    [directory, chatList]
  );

  const title = useMemo(() => (selectedChat ? selectedChat.user.email : "Chat"), [selectedChat]);
  const subtitle = selectedChat ? "Direct conversation" : "Conversations and team directory";

  return (
    <ScreenContainer testID="chat-screen">
      <Header
        action={
          <Button
            disabled={loading || sending}
            label="Reload"
            onPress={reloadCurrentView}
            size="sm"
            testID="chat-reload-button"
            variant="outline"
          />
        }
        eyebrow="Inbox"
        leading={
          selectedChat ? (
            <Button
              label="Back"
              onPress={() => {
                setSelectedChat(null);
                setMessages([]);
                setDraft("");
                setError(null);
                setBackgroundError(null);
              }}
              size="sm"
              testID="chat-back-button"
              variant="ghost"
            />
          ) : null
        }
        style={styles.headerShell}
        subtitle={subtitle}
        title={title}
      />

      {error ? (
        <ErrorView actionLabel="Retry" compact message={error} onAction={() => void reloadCurrentView()} style={styles.errorWrap} />
      ) : null}

      {!error && backgroundError ? (
        <Notice
          description="The chat UI is still usable, but the latest background refresh did not complete."
          style={styles.warnWrap}
          title={backgroundError}
          tone="warning"
        />
      ) : null}

      {loading ? (
        <LoadingView
          fullScreen
          label={selectedChat ? "Loading conversation..." : "Loading chat workspace..."}
          style={styles.loaderWrap}
        />
      ) : selectedChat ? (
        <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.chatWrap}>
          <FlatList
            contentContainerStyle={styles.messageList}
            data={messages}
            keyExtractor={(item) => item.id}
            ListEmptyComponent={
              <EmptyView
                message="Start with a quick hello to open the conversation."
                title="No messages yet"
              />
            }
            renderItem={({ item }) => {
              const mine = item.senderUserId === currentUserId;
              return <MessageItem content={item.content} mine={mine} />;
            }}
          />

          <View style={styles.composer}>
            <Input
              containerStyle={styles.inputWrap}
              onChangeText={setDraft}
              placeholder="Write a message..."
              style={styles.input}
              testID="chat-message-input"
              value={draft}
            />
            <Button
              disabled={loading}
              label="Send"
              loading={sending}
              onPress={onSend}
              size="sm"
              testID="chat-send-button"
            />
          </View>
        </KeyboardAvoidingView>
      ) : (
        <ScrollView contentContainerStyle={styles.scrollContent}>
          <Section eyebrow="Inbox" title="Recent conversations">
            {chatList.length === 0 ? (
              <EmptyView message="New conversations will appear here as soon as you start messaging." title="No conversations yet" />
            ) : null}
            {chatList.map((chat) => (
              <ListItem
                key={chat.user.id}
                disabled={loading || sending}
                leading={<Avatar name={chat.user.email} size={40} />}
                onPress={() => void openChat(chatSummaryToTarget(chat))}
                style={loading || sending ? styles.chatCardDisabled : null}
                subtitle={chat.lastMessage?.content ?? "Open conversation"}
                testID={`chat-open-${chat.user.id}`}
                title={chat.user.email}
              />
            ))}
          </Section>

          <Section eyebrow="Directory" title="Start a new conversation">
            {availableContacts.length === 0 ? (
              <EmptyView message="Invite more teammates to unlock new conversations." title="No additional members available" />
            ) : null}
            {availableContacts.map((user) => (
              <ListItem
                key={user.id}
                disabled={loading || sending}
                leading={<Avatar name={user.email} size={40} />}
                onPress={() => void openChat({ user })}
                style={loading || sending ? styles.chatCardDisabled : null}
                subtitle="Message this member"
                title={user.email}
                variant="muted"
              />
            ))}
          </Section>
        </ScrollView>
      )}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  headerShell: {
    marginBottom: spacing.md,
  },
  scrollContent: {
    paddingBottom: spacing.xxxl
  },
  chatCardDisabled: {
    opacity: 0.7
  },
  errorWrap: {
    marginBottom: spacing.sm,
  },
  warnWrap: {
    marginBottom: spacing.sm,
  },
  loaderWrap: {
    flex: 1
  },
  chatWrap: {
    flex: 1
  },
  messageList: {
    paddingBottom: spacing.md
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.sm,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border
  },
  inputWrap: {
    flex: 1
  },
  input: {
    minHeight: 48
  }
});
