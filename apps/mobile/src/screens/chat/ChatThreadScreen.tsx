import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../../api/client";
import { clearConversation, markRead, sendMessage } from "../../api/dating";
import type { Message } from "../../api/types";
import { useActiveChat } from "../../hooks/activeChat";
import { clearUnread, qk, removeMessage, upsertMessage, useMessages } from "../../hooks/queries";
import { useCurrentUser } from "../../hooks/useAuth";
import { confirm } from "../../lib/confirm";
import { formatDayLabel, formatMessageTime, sameDay } from "../../lib/format";
import type { MainScreenProps } from "../../navigation/types";
import { Avatar, EmptyState, IconButton, LoadingState, Text, colors, radii, showToast, spacing } from "../../ui";

const MAX_LENGTH = 2000;

export default function ChatThreadScreen({ navigation, route }: MainScreenProps<"ChatThread">) {
  const { userId, firstName, photoUrl } = route.params;
  const me = useCurrentUser();
  const client = useQueryClient();
  const activeChat = useActiveChat();
  const thread = useMessages(userId);
  const [draft, setDraft] = useState("");
  const lastReadSent = useRef<string | null>(null);

  useFocusEffect(
    useCallback(() => {
      activeChat.current = userId;
      return () => {
        activeChat.current = null;
      };
    }, [activeChat, userId])
  );

  const pages = thread.data?.pages;
  // Newest first, for the inverted list.
  const messages = useMemo(() => (pages ?? []).flatMap((p) => [...p.messages].reverse()), [pages]);
  const otherLastReadAt = pages?.[0]?.otherLastReadAt;
  const newestFromOther = messages.find((m) => m.senderUserId !== me.id);

  // Mark as read whenever a new incoming message is on screen.
  useEffect(() => {
    if (!newestFromOther || lastReadSent.current === newestFromOther.id) {
      return;
    }
    lastReadSent.current = newestFromOther.id;
    clearUnread(client, userId);
    markRead(userId).catch(() => {
      lastReadSent.current = null;
    });
  }, [client, newestFromOther, userId]);

  const send = async () => {
    const content = draft.trim();
    if (!content) {
      return;
    }
    setDraft("");
    const tempId = `pending-${Date.now()}`;
    const optimistic: Message = {
      id: tempId,
      conversationId: "",
      senderUserId: me.id,
      recipientUserId: userId,
      content,
      createdAt: new Date().toISOString(),
      pending: true
    };
    upsertMessage(client, userId, optimistic);
    try {
      const saved = await sendMessage(userId, content);
      upsertMessage(client, userId, saved, tempId);
      void client.invalidateQueries({ queryKey: qk.chats });
    } catch (err) {
      removeMessage(client, userId, tempId);
      setDraft(content);
      showToast(errorMessage(err), "error");
    }
  };

  const clear = async () => {
    if (
      !(await confirm(
        "Effacer la conversation ?",
        `L'historique sera masqué pour vous uniquement. ${firstName} le conservera.`,
        "Effacer",
        true
      ))
    ) {
      return;
    }
    try {
      await clearConversation(userId);
      await client.invalidateQueries({ queryKey: qk.messages(userId) });
      void client.invalidateQueries({ queryKey: qk.chats });
      showToast("Conversation effacée");
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const unavailable = thread.isError && thread.error instanceof ApiError && thread.error.status === 404;
  const myLatest = messages.find((m) => m.senderUserId === me.id);

  return (
    <SafeAreaView edges={["top", "left", "right", "bottom"]} style={styles.safe} testID="chat-thread">
      <View style={styles.header}>
        <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} size={40} />
        <Pressable
          accessibilityLabel={`Voir le profil de ${firstName}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("ProfileDetail", { userId, context: "match" })}
          style={styles.who}
        >
          <Avatar name={firstName} size={40} url={photoUrl} />
          <Text numberOfLines={1} variant="heading">
            {firstName}
          </Text>
        </Pressable>
        <IconButton icon="trash-outline" label="Effacer la conversation" onPress={clear} size={40} />
      </View>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={8} style={styles.flex}>
        {thread.isPending ? (
          <LoadingState />
        ) : unavailable ? (
          <EmptyState
            icon="chatbubble-outline"
            message="Ce match a pris fin ou le profil n'est plus disponible."
            title="Conversation indisponible"
          />
        ) : thread.isError ? (
          <EmptyState
            actionLabel="Réessayer"
            icon="cloud-offline-outline"
            message={errorMessage(thread.error)}
            onAction={() => void thread.refetch()}
            title="Chargement impossible"
          />
        ) : (
          <FlatList
            contentContainerStyle={styles.list}
            data={messages}
            inverted={messages.length > 0}
            keyExtractor={(m) => m.id}
            ListEmptyComponent={
              <EmptyState
                icon="sparkles-outline"
                message={`Vous avez matché avec ${firstName}. Un simple « bonjour » suffit souvent !`}
                title="Brisez la glace"
              />
            }
            ListFooterComponent={thread.isFetchingNextPage ? <ActivityIndicator color={colors.accent} style={styles.more} /> : null}
            onEndReached={() => {
              if (thread.hasNextPage && !thread.isFetchingNextPage) {
                void thread.fetchNextPage();
              }
            }}
            onEndReachedThreshold={0.3}
            renderItem={({ item, index }) => {
              const mine = item.senderUserId === me.id;
              const older = messages[index + 1];
              const showDay = !older || !sameDay(older.createdAt, item.createdAt);
              const isMyLatest = myLatest?.id === item.id;
              const read = mine && otherLastReadAt && new Date(otherLastReadAt) >= new Date(item.createdAt);
              return (
                <View>
                  {showDay ? (
                    <Text align="center" style={styles.day} tone="subtle" variant="caption">
                      {formatDayLabel(item.createdAt)}
                    </Text>
                  ) : null}
                  <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
                    <View
                      style={[styles.bubble, mine ? styles.mine : styles.theirs, item.pending ? styles.pending : null]}
                      testID={mine ? "message-mine" : "message-theirs"}
                    >
                      <Text style={mine ? styles.mineText : undefined}>{item.content}</Text>
                      <Text style={[styles.time, mine ? styles.mineTime : null]} variant="caption">
                        {formatMessageTime(item.createdAt)}
                      </Text>
                    </View>
                  </View>
                  {isMyLatest ? (
                    <Text style={styles.status} testID="message-status" tone="subtle" variant="caption">
                      {item.pending ? "Envoi…" : read ? "Lu" : "Envoyé"}
                    </Text>
                  ) : null}
                </View>
              );
            }}
          />
        )}
        {!unavailable ? (
          <View style={styles.composer}>
            <TextInput
              accessibilityLabel="Votre message"
              maxLength={MAX_LENGTH}
              multiline
              onChangeText={setDraft}
              placeholder={`Écrire à ${firstName}…`}
              placeholderTextColor={colors.textSubtle}
              selectionColor={colors.accent}
              style={styles.input}
              testID="chat-input"
              value={draft}
            />
            <IconButton
              background={draft.trim() ? colors.accent : colors.surfaceRaised}
              color={draft.trim() ? colors.textOnAccent : colors.textSubtle}
              disabled={!draft.trim()}
              icon="arrow-up"
              label="Envoyer"
              onPress={send}
              testID="chat-send"
            />
          </View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: colors.background
  },
  flex: {
    flex: 1,
    width: "100%",
    maxWidth: 720,
    alignSelf: "center"
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  who: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  list: {
    flexGrow: 1,
    padding: spacing.md,
    gap: spacing.xxs
  },
  more: {
    marginVertical: spacing.md
  },
  day: {
    marginVertical: spacing.sm
  },
  bubbleRow: {
    flexDirection: "row",
    justifyContent: "flex-start"
  },
  mineRow: {
    justifyContent: "flex-end"
  },
  bubble: {
    maxWidth: "80%",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderRadius: radii.lg,
    gap: 2
  },
  mine: {
    backgroundColor: colors.accent,
    borderBottomRightRadius: 6
  },
  theirs: {
    backgroundColor: colors.surfaceRaised,
    borderBottomLeftRadius: 6
  },
  pending: {
    opacity: 0.6
  },
  mineText: {
    color: colors.textOnAccent
  },
  time: {
    alignSelf: "flex-end",
    fontSize: 11,
    color: colors.textSubtle
  },
  mineTime: {
    color: "rgba(27, 20, 32, 0.65)"
  },
  status: {
    alignSelf: "flex-end",
    marginTop: 2,
    marginRight: spacing.xs
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 140,
    paddingHorizontal: spacing.md,
    paddingTop: 11,
    paddingBottom: 11,
    borderRadius: radii.lg,
    backgroundColor: colors.surfaceRaised,
    color: colors.text,
    fontSize: 16
  }
});
