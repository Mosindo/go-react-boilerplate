import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { hideConversation, markConversationRead, sendMessage } from "../api/chat";
import { ApiError, errorMessage } from "../api/client";
import { unmatch } from "../api/discovery";
import { getProfile } from "../api/profile";
import { blockUser } from "../api/safety";
import type { Message, MessagesPage } from "../api/types";
import { ActionSheet } from "../components/ActionSheet";
import { ConfirmModal } from "../components/ConfirmModal";
import { NavHeader } from "../components/NavHeader";
import { ReportModal } from "../components/ReportModal";
import { Screen } from "../components/Screen";
import { queryKeys, useConversations, useMessages } from "../hooks/queries";
import { useCurrentUser } from "../hooks/useAuth";
import { useRealtime } from "../hooks/useRealtime";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Button, Loader, Text, colors, radii, spacing } from "../shared/ui";
import { formatClock } from "../utils/format";

type Props = NativeStackScreenProps<MainStackParamList, "Conversation">;
type Dialog = "menu" | "hide" | "unmatch" | "block" | "report" | null;

const MAX_LENGTH = 2000;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId } = route.params;
  const me = useCurrentUser();
  const client = useQueryClient();
  const { setActiveConversation } = useRealtime();
  const [draft, setDraft] = useState("");
  const [dialog, setDialog] = useState<Dialog>(null);

  const messagesQuery = useMessages(conversationId);
  const conversations = useConversations();
  const summary = conversations.data?.pages.flatMap((p) => p.conversations).find((c) => c.id === conversationId);
  const person = useQuery({ queryKey: queryKeys.person(userId), queryFn: () => getProfile(userId), retry: false });

  const name = route.params.name || summary?.user.firstName || person.data?.firstName || "Conversation";
  const matchId = route.params.matchId ?? summary?.matchId;

  const messages = useMemo(() => messagesQuery.data?.pages.flatMap((p) => p.messages) ?? [], [messagesQuery.data]);
  const otherLastReadAt = messagesQuery.data?.pages[0]?.otherLastReadAt ?? null;
  const lastIncomingId = messages.find((m) => m.senderId !== me.id)?.id;

  // The conversation is gone (unmatched or blocked by the other person).
  const unavailable = messagesQuery.error instanceof ApiError && messagesQuery.error.status === 404;

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      return () => setActiveConversation(null);
    }, [conversationId, setActiveConversation])
  );

  useEffect(() => {
    if (!lastIncomingId && !summary?.unreadCount) {
      return;
    }
    markConversationRead(conversationId)
      .then(() => client.invalidateQueries({ queryKey: queryKeys.conversations }))
      .catch(() => undefined);
  }, [conversationId, lastIncomingId, summary?.unreadCount, client]);

  const send = useMutation({
    mutationFn: (content: string) => sendMessage(conversationId, content),
    onSuccess: (message) => {
      client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (old) => {
        if (!old || old.pages.length === 0 || old.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
          return old;
        }
        const [first, ...rest] = old.pages;
        return { ...old, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
      });
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (error, content) => {
      setDraft((current) => current || content);
      showToast(errorMessage(error, "Message non envoyé."), "error");
    }
  });

  const submit = () => {
    const content = draft.trim();
    if (!content || send.isPending) {
      return;
    }
    setDraft("");
    send.mutate(content);
  };

  const leave = () => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    navigation.navigate("Tabs");
  };
  const fail = (error: unknown) => showToast(errorMessage(error), "error");
  const hide = useMutation({ mutationFn: () => hideConversation(conversationId), onSuccess: leave, onError: fail });
  const end = useMutation({
    mutationFn: async () => {
      if (!matchId) throw new Error("Match introuvable.");
      await unmatch(matchId);
    },
    onSuccess: () => {
      showToast("Match supprimé.", "info");
      leave();
    },
    onError: fail
  });
  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: () => {
      showToast(`${name} est bloqué(e).`, "info");
      leave();
    },
    onError: fail
  });

  const lastOwnId = messages.find((m) => m.senderId === me.id)?.id;
  const renderItem = ({ item }: { item: Message }) => {
    const mine = item.senderId === me.id;
    const read =
      mine &&
      item.id === lastOwnId &&
      otherLastReadAt !== null &&
      new Date(otherLastReadAt) >= new Date(item.createdAt);
    return (
      <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
        <View
          accessible
          style={[styles.bubble, mine ? styles.bubbleMine : styles.bubbleTheirs]}
          testID={mine ? "message-mine" : "message-theirs"}
        >
          <Text tone={mine ? "inverse" : "default"}>{item.content}</Text>
          <Text style={styles.time} tone={mine ? "inverse" : "muted"} variant="caption">
            {formatClock(item.createdAt)}
            {read ? " · Lu" : ""}
          </Text>
        </View>
      </View>
    );
  };

  return (
    <Screen
      contentStyle={styles.content}
      edges={["top", "left", "right", "bottom"]}
      header={
        <NavHeader
          onBack={() => navigation.goBack()}
          right={
            <Pressable
              accessibilityLabel="Plus d'options"
              accessibilityRole="button"
              hitSlop={10}
              onPress={() => setDialog("menu")}
              style={styles.more}
              testID="chat-menu"
            >
              <Text style={styles.moreGlyph} weight="bold">
                ⋯
              </Text>
            </Pressable>
          }
          title={name}
        />
      }
      testID="chat-screen"
    >
      {unavailable ? (
        <View style={styles.gone}>
          <Text variant="heading" weight="bold">
            Conversation indisponible
          </Text>
          <Text tone="muted">Ce match n'existe plus.</Text>
          <Button label="Retour" onPress={leave} />
        </View>
      ) : (
        <>
          {messagesQuery.isLoading ? <Loader fullScreen label="Chargement…" /> : null}
          {messagesQuery.data ? (
            <FlatList
              ListEmptyComponent={
                <Text style={styles.empty} tone="muted">
                  C'est un match ! Envoyez le premier message à {name}.
                </Text>
              }
              contentContainerStyle={styles.list}
              data={messages}
              inverted
              keyExtractor={(m) => m.id}
              onEndReached={() => {
                if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage) void messagesQuery.fetchNextPage();
              }}
              onEndReachedThreshold={0.3}
              renderItem={renderItem}
            />
          ) : null}
          <View style={styles.composer}>
            <TextInput
              accessibilityLabel="Votre message"
              maxLength={MAX_LENGTH}
              multiline
              onChangeText={setDraft}
              onKeyPress={(event) => {
                // Enter sends on web; Shift+Enter inserts a new line.
                const native = event.nativeEvent as unknown as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  event.preventDefault?.();
                  submit();
                }
              }}
              placeholder="Écrire un message…"
              placeholderTextColor={colors.textSubtle}
              style={styles.input}
              testID="chat-input"
              value={draft}
            />
            <Button
              disabled={!draft.trim()}
              label="Envoyer"
              loading={send.isPending}
              onPress={submit}
              size="sm"
              testID="chat-send"
            />
          </View>
        </>
      )}

      <ActionSheet
        actions={[
          {
            label: "Voir le profil",
            onPress: () => navigation.navigate("Person", { userId, matchId, conversationId }),
            testID: "menu-profile"
          },
          { label: "Masquer la conversation", onPress: () => setDialog("hide"), testID: "menu-hide" },
          {
            label: "Supprimer le match",
            onPress: () => setDialog("unmatch"),
            destructive: true,
            testID: "menu-unmatch"
          },
          { label: "Bloquer", onPress: () => setDialog("block"), destructive: true, testID: "menu-block" },
          { label: "Signaler", onPress: () => setDialog("report"), destructive: true, testID: "menu-report" }
        ]}
        onClose={() => setDialog((d) => (d === "menu" ? null : d))}
        visible={dialog === "menu"}
      />
      <ConfirmModal
        confirmLabel="Masquer"
        loading={hide.isPending}
        message="La conversation disparaît de votre liste. Elle réapparaîtra si cette personne vous écrit."
        onCancel={() => setDialog(null)}
        onConfirm={() => hide.mutate()}
        title="Masquer la conversation ?"
        visible={dialog === "hide"}
      />
      <ConfirmModal
        confirmLabel="Supprimer le match"
        destructive
        loading={end.isPending}
        message={`Vous ne pourrez plus échanger avec ${name} et les messages seront supprimés pour vous deux.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => end.mutate()}
        title="Supprimer le match ?"
        visible={dialog === "unmatch"}
      />
      <ConfirmModal
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        message={`${name} ne pourra plus vous voir ni vous écrire, et le match sera supprimé.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => block.mutate()}
        title={`Bloquer ${name} ?`}
        visible={dialog === "block"}
      />
      <ReportModal
        firstName={name}
        onClose={() => setDialog(null)}
        onDone={leave}
        userId={userId}
        visible={dialog === "report"}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: 0, gap: 0 },
  list: { padding: spacing.lg, gap: spacing.sm },
  empty: { textAlign: "center", padding: spacing.xl, transform: [{ scaleY: -1 }] },
  bubbleRow: { flexDirection: "row" },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: {
    maxWidth: "80%",
    borderRadius: radii.lg,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    gap: 2
  },
  bubbleMine: { backgroundColor: colors.primary, borderBottomRightRadius: 4 },
  bubbleTheirs: { backgroundColor: colors.surfaceSubtle, borderBottomLeftRadius: 4 },
  time: { alignSelf: "flex-end", opacity: 0.8 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    backgroundColor: colors.backgroundElevated,
    color: colors.text,
    fontSize: 15
  },
  more: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  moreGlyph: { fontSize: 26, lineHeight: 30 },
  gone: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
