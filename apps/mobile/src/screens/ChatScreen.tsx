import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useFocusEffect, useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { blockUser, hideConversation, markConversationRead, sendMessage, unmatch } from "../api/platform";
import type { Message } from "../api/types";
import { ActionSheet } from "../components/ActionSheet";
import { appendMessage, queryKeys, useMessages } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { useRealtime } from "../hooks/useRealtime";
import { errorCodeOf, errorMessage } from "../i18n/errors";
import { formatDayLabel, isSameDay } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Chip, MessageItem, Text, radii, spacing, type MessageStatus } from "../shared/ui";
import { useTheme } from "../theme";
import { confirm } from "../lib/confirm";

const MAX_LENGTH = 2000;
const ICEBREAKERS = [
  "Salut ! Comment se passe ta semaine ?",
  "Quelle est ta dernière belle découverte ?",
  "Plutôt café ou balade pour une première rencontre ?"
];

type Pending = { tempId: string; body: string; createdAt: string; failed: boolean };

type Row = { kind: "message"; message: Message; status?: MessageStatus; pending?: Pending } | { kind: "pending"; pending: Pending };

export default function ChatScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { params } = useRoute<RouteProp<RootStackParamList, "Thread">>();
  const { conversationId, peerId, peerName } = params;
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { setActiveConversation } = useRealtime();
  const messagesQuery = useMessages(conversationId);

  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<Pending[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const sending = useRef(false);

  const pages = messagesQuery.data?.pages;
  const messages = useMemo(() => pages?.flatMap((page) => page.messages) ?? [], [pages]);
  const peerLastReadAt = pages?.[0]?.peerLastReadAt;
  const newestPeerMessageId = messages.find((m) => m.senderId !== user?.id)?.id;

  // Tell the server (and the other person) what has been seen, now and whenever the peer writes.
  const markRead = useCallback(() => {
    void markConversationRead(conversationId)
      .then(() => queryClient.invalidateQueries({ queryKey: queryKeys.chats }))
      .then(() => queryClient.invalidateQueries({ queryKey: queryKeys.notifications }))
      .catch(() => undefined);
  }, [conversationId, queryClient]);

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      markRead();
      return () => setActiveConversation(null);
    }, [conversationId, markRead, setActiveConversation])
  );

  useEffect(() => {
    if (newestPeerMessageId) {
      markRead();
    }
  }, [markRead, newestPeerMessageId]);

  const rows = useMemo<Row[]>(() => {
    const lastMineId = messages.find((m) => m.senderId === user?.id)?.id;
    const server: Row[] = messages.map((message) => {
      if (message.id !== lastMineId) {
        return { kind: "message", message } as Row;
      }
      const read = peerLastReadAt !== undefined && new Date(peerLastReadAt) >= new Date(message.createdAt);
      return { kind: "message", message, status: read ? "read" : "sent" } as Row;
    });
    return [...[...pending].reverse().map((p): Row => ({ kind: "pending", pending: p })), ...server];
  }, [messages, pending, peerLastReadAt, user?.id]);

  const send = async (body: string, tempId?: string) => {
    if (sending.current && !tempId) {
      return;
    }
    const text = body.trim();
    if (!text) {
      return;
    }
    const id = tempId ?? `tmp-${Date.now()}`;
    if (!tempId) {
      setDraft("");
      setPending((current) => [...current, { tempId: id, body: text, createdAt: new Date().toISOString(), failed: false }]);
    } else {
      setPending((current) => current.map((p) => (p.tempId === id ? { ...p, failed: false } : p)));
    }
    sending.current = true;
    try {
      const sent = await sendMessage(conversationId, text);
      appendMessage(queryClient, sent);
      setPending((current) => current.filter((p) => p.tempId !== id));
      void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
    } catch (error) {
      if (errorCodeOf(error) === "conversation_not_found") {
        showError("Cette conversation n'existe plus.");
        navigation.goBack();
        return;
      }
      setPending((current) => current.map((p) => (p.tempId === id ? { ...p, failed: true } : p)));
      showError(errorMessage(error, "Message non envoyé. Touchez-le pour réessayer."));
    } finally {
      sending.current = false;
    }
  };

  const leaveAfter = async (task: () => Promise<unknown>, done: string) => {
    try {
      await task();
      queryClient.removeQueries({ queryKey: queryKeys.messages(conversationId) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.chats });
      await queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
      showSuccess(done);
      navigation.goBack();
    } catch (error) {
      showError(errorMessage(error));
    }
  };

  const menuActions = [
    { label: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId: peerId, conversationId }) },
    {
      label: "Supprimer la conversation",
      onPress: () =>
        confirm({
          title: "Supprimer la conversation ?",
          message: `Elle disparaîtra de votre liste. ${peerName} la conservera, et elle réapparaîtra si un nouveau message arrive.`,
          confirmLabel: "Supprimer",
          onConfirm: () => void leaveAfter(() => hideConversation(conversationId), "Conversation supprimée.")
        })
    },
    {
      label: `Annuler le match avec ${peerName}`,
      destructive: true,
      onPress: () =>
        confirm({
          title: "Annuler le match ?",
          message: "La conversation sera supprimée pour vous deux. Vous ne vous reverrez pas dans la découverte.",
          confirmLabel: "Annuler le match",
          destructive: true,
          onConfirm: () => void leaveAfter(() => unmatch(peerId), "Match annulé.")
        })
    },
    { label: "Signaler", destructive: true, onPress: () => navigation.navigate("Report", { userId: peerId, name: peerName }) },
    {
      label: `Bloquer ${peerName}`,
      destructive: true,
      onPress: () =>
        confirm({
          title: `Bloquer ${peerName} ?`,
          message: "Vous ne verrez plus cette personne et elle ne pourra plus vous voir ni vous écrire. La conversation sera supprimée.",
          confirmLabel: "Bloquer",
          destructive: true,
          onConfirm: () => void leaveAfter(() => blockUser(peerId), `${peerName} est bloqué(e).`)
        })
    }
  ];

  const renderItem = ({ item, index }: { item: Row; index: number }) => {
    const message = item.kind === "message" ? item.message : null;
    const createdAt = message?.createdAt ?? item.pending!.createdAt;
    const older = rows[index + 1];
    const olderAt = older ? (older.kind === "message" ? older.message.createdAt : older.pending.createdAt) : null;
    const showDay = !olderAt || !isSameDay(createdAt, olderAt);
    const senderId = message?.senderId ?? user?.id;
    const olderSender = older ? (older.kind === "message" ? older.message.senderId : user?.id) : null;
    const grouped = !showDay && olderSender === senderId;

    return (
      <View>
        {showDay ? (
          <Text style={{ textAlign: "center", marginTop: spacing.lg }} tone="subtle" variant="caption" weight="semibold">
            {formatDayLabel(createdAt).toUpperCase()}
          </Text>
        ) : null}
        {item.kind === "message" ? (
          <MessageItem
            body={item.message.body}
            createdAt={item.message.createdAt}
            grouped={grouped}
            mine={item.message.senderId === user?.id}
            status={item.status}
          />
        ) : (
          <Pressable
            accessibilityHint={item.pending.failed ? "Touchez pour réessayer l'envoi" : undefined}
            disabled={!item.pending.failed}
            onPress={() => void send(item.pending.body, item.pending.tempId)}
          >
            <MessageItem
              body={item.pending.body}
              createdAt={item.pending.createdAt}
              grouped={grouped}
              mine
              status={item.pending.failed ? "failed" : "sending"}
            />
          </Pressable>
        )}
      </View>
    );
  };

  const missing = messagesQuery.isError && errorCodeOf(messagesQuery.error) === "conversation_not_found";
  const empty = !messagesQuery.isLoading && rows.length === 0;

  return (
    <SafeAreaLayout edges={["top", "left", "right", "bottom"]}>
      <View style={{ paddingHorizontal: spacing.lg }}>
        <Header
          onBack={() => navigation.goBack()}
          right={
            <Pressable
              accessibilityLabel="Plus d'options"
              accessibilityRole="button"
              hitSlop={8}
              onPress={() => setMenuOpen(true)}
              style={{ width: 44, height: 44, alignItems: "center", justifyContent: "center" }}
              testID="chat-menu"
            >
              <Ionicons color={palette.text} name="ellipsis-horizontal" size={24} />
            </Pressable>
          }
          title={peerName}
        />
      </View>

      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={{ flex: 1 }}>
        {missing ? (
          <ErrorView message="Cette conversation n'existe plus." onRetry={() => navigation.goBack()} title="Conversation introuvable" />
        ) : messagesQuery.isLoading ? (
          <LoadingView fullScreen />
        ) : messagesQuery.isError && rows.length === 0 ? (
          <ErrorView error={messagesQuery.error} onRetry={() => void messagesQuery.refetch()} />
        ) : (
          <FlatList
            contentContainerStyle={{
              paddingHorizontal: spacing.lg,
              paddingBottom: spacing.md,
              flexGrow: 1,
              justifyContent: empty ? "center" : "flex-start"
            }}
            data={rows}
            inverted={!empty}
            keyExtractor={(row) => (row.kind === "message" ? row.message.id : row.pending.tempId)}
            keyboardDismissMode="interactive"
            keyboardShouldPersistTaps="handled"
            ListEmptyComponent={
              <View style={{ alignItems: "center", gap: spacing.md }} testID="chat-empty">
                <Ionicons color={palette.primary} name="heart" size={40} />
                <Text style={{ textAlign: "center" }} variant="heading" weight="bold">
                  C&apos;est un match avec {peerName} !
                </Text>
                <Text style={{ textAlign: "center" }} tone="muted">
                  Brisez la glace avec un premier message.
                </Text>
                <View style={{ gap: spacing.sm, alignItems: "center" }}>
                  {ICEBREAKERS.map((text) => (
                    <Chip key={text} label={text} onPress={() => setDraft(text)} />
                  ))}
                </View>
              </View>
            }
            ListFooterComponent={messagesQuery.isFetchingNextPage ? <LoadingView /> : null}
            onEndReached={() => {
              if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage) {
                void messagesQuery.fetchNextPage();
              }
            }}
            onEndReachedThreshold={0.4}
            renderItem={renderItem}
            testID="chat-screen"
          />
        )}

        {!missing ? (
          <View
            style={{
              flexDirection: "row",
              alignItems: "flex-end",
              gap: spacing.sm,
              paddingHorizontal: spacing.lg,
              paddingVertical: spacing.sm,
              borderTopWidth: 1,
              borderTopColor: palette.border,
              backgroundColor: palette.background
            }}
          >
            <View
              style={{
                flex: 1,
                minHeight: 48,
                maxHeight: 140,
                borderRadius: radii.xl,
                borderWidth: 1.5,
                borderColor: palette.border,
                backgroundColor: palette.surface,
                justifyContent: "center",
                paddingHorizontal: spacing.lg
              }}
            >
              <TextInputWithAutosize onChangeText={setDraft} value={draft} />
            </View>
            <Pressable
              accessibilityLabel="Envoyer"
              accessibilityRole="button"
              accessibilityState={{ disabled: !draft.trim() }}
              disabled={!draft.trim()}
              onPress={() => void send(draft)}
              style={({ pressed }) => ({
                width: 48,
                height: 48,
                borderRadius: 24,
                alignItems: "center",
                justifyContent: "center",
                backgroundColor: palette.primary,
                opacity: !draft.trim() ? 0.4 : pressed ? 0.8 : 1
              })}
              testID="chat-send-button"
            >
              <Ionicons color={palette.onPrimary} name="arrow-up" size={24} />
            </Pressable>
          </View>
        ) : null}
      </KeyboardAvoidingView>

      <ActionSheet actions={menuActions} onClose={() => setMenuOpen(false)} title={peerName} visible={menuOpen} />
    </SafeAreaLayout>
  );
}

function TextInputWithAutosize({ onChangeText, value }: { onChangeText: (text: string) => void; value: string }) {
  const palette = useTheme();
  return (
    <TextInput
      accessibilityLabel="Votre message"
      allowFontScaling
      maxLength={MAX_LENGTH}
      maxFontSizeMultiplier={1.4}
      multiline
      onChangeText={onChangeText}
      placeholder="Écrire un message…"
      placeholderTextColor={palette.textSubtle}
      style={{ color: palette.text, fontSize: 16, maxHeight: 120, paddingVertical: Platform.OS === "ios" ? 10 : 6 }}
      testID="chat-message-input"
      value={value}
    />
  );
}
