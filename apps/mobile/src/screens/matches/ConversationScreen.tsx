import React, { useCallback, useEffect, useMemo, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { ApiError } from "../../api/client";
import { chatApi } from "../../api/social";
import type { Message } from "../../api/types";
import { SafetySheet } from "../../components/SafetySheet";
import { keys, upsertMessage, useMatch, useMessages } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { subscribeRealtime } from "../../hooks/useRealtime";
import type { AppScreenProps } from "../../navigation/types";
import { colors, radii, spacing, typography } from "../../theme/tokens";
import { Avatar, EmptyState, ErrorState, IconButton, LoadingState, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { messageTime } from "../../utils/format";
import { newId } from "../../utils/id";

const MAX_LENGTH = 2000;

export default function ConversationScreen({ navigation, route }: AppScreenProps<"Conversation">) {
  const { matchId } = route.params;
  const { user } = useAuth();
  const myId = user?.id ?? "";
  const qc = useQueryClient();
  const match = useMatch(matchId);
  const messages = useMessages(matchId);
  const [draft, setDraft] = useState("");
  const [safetyOpen, setSafetyOpen] = useState(false);

  const items = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const lastOwnRead = items.find((m) => m.senderId === myId && !m.pending);

  const markRead = useCallback(() => {
    chatApi
      .markRead(matchId)
      .then(() => {
        void qc.invalidateQueries({ queryKey: keys.matches });
        void qc.invalidateQueries({ queryKey: keys.notifications });
      })
      .catch(() => undefined);
  }, [matchId, qc]);

  useFocusEffect(
    useCallback(() => {
      markRead();
      return subscribeRealtime((ev) => {
        if (ev.type === "message.created" && ev.data.matchId === matchId && ev.data.senderId !== myId) {
          markRead();
        }
        if (ev.type === "match.removed" && ev.data.matchId === matchId) {
          showToast("Cette conversation n'est plus disponible.", "info");
          navigation.navigate("Tabs", { screen: "Matches" });
        }
      });
    }, [markRead, matchId, myId, navigation])
  );

  useEffect(() => {
    if (match.error instanceof ApiError && match.error.status === 404) {
      qc.removeQueries({ queryKey: keys.messages(matchId) });
    }
  }, [match.error, matchId, qc]);

  const setCache = (m: Message) => qc.setQueryData(keys.messages(matchId), (cache: Parameters<typeof upsertMessage>[0]) => upsertMessage(cache, m));

  const deliver = async (body: string, clientMessageId: string) => {
    const optimistic: Message = {
      id: `local-${clientMessageId}`,
      matchId,
      senderId: myId,
      body,
      clientMessageId,
      createdAt: new Date().toISOString(),
      readAt: null,
      pending: true
    };
    setCache(optimistic);
    try {
      const saved = await chatApi.send(matchId, body, clientMessageId);
      setCache(saved);
      void qc.invalidateQueries({ queryKey: keys.matches });
    } catch (err) {
      setCache({ ...optimistic, pending: false, failed: true });
      showToast(errorMessage(err, "Message non envoyé. Touche-le pour réessayer."), "error");
    }
  };

  const send = () => {
    const body = draft.trim();
    if (!body) {
      return;
    }
    setDraft("");
    void deliver(body, newId());
  };

  if (match.isLoading) {
    return (
      <SafeAreaView style={styles.root}>
        <LoadingState />
      </SafeAreaView>
    );
  }
  if (match.error || !match.data) {
    const gone = match.error instanceof ApiError && match.error.status === 404;
    return (
      <SafeAreaView style={styles.root}>
        <View style={styles.headerBar}>
          <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} background={colors.transparent} />
        </View>
        {gone ? (
          <EmptyState icon="chatbubbles-outline" title="Conversation indisponible" message="Ce match a pris fin." />
        ) : (
          <ErrorState message={errorMessage(match.error)} onRetry={() => match.refetch()} />
        )}
      </SafeAreaView>
    );
  }

  const other = match.data.user;
  return (
    <SafeAreaView style={styles.root} edges={["top", "bottom"]} testID="conversation-screen">
      <View style={styles.headerBar}>
        <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} background={colors.transparent} />
        <Pressable
          style={styles.headerPerson}
          onPress={() => navigation.navigate("ProfileDetail", { userId: other.userId, card: other, matchId })}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${other.firstName}`}
        >
          <Avatar uri={other.photos[0]?.url} name={other.firstName} size={40} />
          <Text variant="heading" numberOfLines={1}>
            {other.firstName}
          </Text>
        </Pressable>
        <IconButton
          icon="ellipsis-horizontal"
          label="Options de sécurité"
          onPress={() => setSafetyOpen(true)}
          background={colors.transparent}
          testID="conversation-safety"
        />
      </View>

      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={8}>
        {messages.isLoading ? (
          <LoadingState />
        ) : (
          <FlatList
            data={items}
            inverted
            keyExtractor={(m) => m.clientMessageId ?? m.id}
            contentContainerStyle={styles.messages}
            onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && messages.fetchNextPage()}
            onEndReachedThreshold={0.3}
            ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={colors.accent} /> : null}
            ListEmptyComponent={
              <View style={styles.emptyChat}>
                <Avatar uri={other.photos[0]?.url} name={other.firstName} size={96} ring />
                <Text variant="heading" align="center">
                  Vous vous plaisez mutuellement
                </Text>
                <Text tone="muted" align="center">
                  Lance la conversation avec {other.firstName}. Un détail de son profil est souvent un bon début.
                </Text>
              </View>
            }
            renderItem={({ item }) => (
              <Bubble
                message={item}
                mine={item.senderId === myId}
                showRead={item === lastOwnRead && !!item.readAt}
                onRetry={() => item.failed && item.clientMessageId && deliver(item.body, item.clientMessageId)}
              />
            )}
          />
        )}
        <View style={styles.composer}>
          <TextInput
            value={draft}
            onChangeText={setDraft}
            placeholder="Écris un message…"
            placeholderTextColor={colors.textSubtle}
            multiline
            maxLength={MAX_LENGTH}
            style={styles.input}
            accessibilityLabel="Message"
            testID="composer-input"
          />
          <IconButton
            icon="arrow-up"
            label="Envoyer"
            onPress={send}
            disabled={!draft.trim()}
            color={colors.textOnAccent}
            background={colors.accent}
            testID="composer-send"
          />
        </View>
      </KeyboardAvoidingView>

      <SafetySheet
        visible={safetyOpen}
        userId={other.userId}
        firstName={other.firstName}
        matchId={matchId}
        onClose={() => setSafetyOpen(false)}
        onDone={() => navigation.navigate("Tabs", { screen: "Matches" })}
      />
    </SafeAreaView>
  );
}

function Bubble({ message, mine, showRead, onRetry }: { message: Message; mine: boolean; showRead: boolean; onRetry: () => void }) {
  return (
    <View style={[styles.bubbleWrap, mine ? styles.right : styles.left]}>
      <Pressable
        onPress={message.failed ? onRetry : undefined}
        style={[styles.bubble, mine ? styles.mine : styles.theirs, message.failed && styles.failed]}
        accessibilityLabel={`${mine ? "Toi" : "Message reçu"} : ${message.body}`}
        testID={mine ? "message-mine" : "message-theirs"}
      >
        <Text style={{ color: mine ? colors.textOnAccent : colors.text }}>{message.body}</Text>
      </Pressable>
      <View style={styles.meta}>
        {message.failed ? <Ionicons name="alert-circle" size={13} color={colors.danger} /> : null}
        <Text variant="caption" tone="subtle" style={styles.metaText}>
          {message.failed ? "Échec · toucher pour renvoyer" : message.pending ? "Envoi…" : messageTime(message.createdAt)}
          {showRead ? " · Lu" : ""}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.background },
  flex: { flex: 1 },
  headerBar: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  headerPerson: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  messages: { padding: spacing.lg, gap: spacing.sm, flexGrow: 1 },
  emptyChat: { alignItems: "center", gap: spacing.md, padding: spacing.xxl, transform: [{ scaleY: -1 }] },
  bubbleWrap: { maxWidth: "82%", gap: 2 },
  left: { alignSelf: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md, borderRadius: radii.xl },
  mine: { backgroundColor: colors.accent, borderBottomRightRadius: radii.sm },
  theirs: { backgroundColor: colors.surfaceRaised, borderBottomLeftRadius: radii.sm },
  failed: { opacity: 0.6 },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, paddingHorizontal: spacing.xs },
  metaText: { fontSize: 11 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border
  },
  input: {
    ...typography.body,
    flex: 1,
    color: colors.text,
    backgroundColor: colors.surface,
    borderRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.md,
    maxHeight: 140
  }
});
