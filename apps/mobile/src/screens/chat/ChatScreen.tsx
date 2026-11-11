import React, { useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Message } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { keys, useMarkRead, useMessages, useSendMessage } from "../../hooks/queries";
import { useRealtime } from "../../hooks/useRealtime";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, IconButton, LoadingState, Text, useFeedback } from "../../ui";
import { formatDayHeader, formatTime, isSameDay } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

export function ChatScreen({ route, navigation }: Props) {
  const { matchId, user: other } = route.params;
  const { colors, radii, spacing } = useTheme();
  const { user } = useAuth();
  const myId = user?.id ?? "";
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const { connected, removedMatchIds } = useRealtime();
  const messages = useMessages(matchId, !connected);
  const send = useSendMessage(matchId, myId);
  const markRead = useMarkRead(matchId);
  const [draft, setDraft] = useState("");

  const hide = useMutation({
    mutationFn: () => chatApi.hide(matchId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      navigation.goBack();
    }
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${other.firstName}`}
          onPress={() => navigation.navigate("ProfileDetail", { userId: other.userId, card: other, matchId })}
          style={styles.headerTitle}
        >
          <Avatar url={other.photos[0]?.url} size={34} />
          <Text variant="label">{other.firstName}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <IconButton
          icon="ellipsis-horizontal"
          label="Options de la conversation"
          onPress={async () => {
            const ok = await confirm({
              title: "Supprimer la conversation ?",
              message:
                "Elle disparaît de ta liste (sans prévenir l'autre personne) et réapparaîtra si un nouveau message arrive. Pour bloquer ou signaler, ouvre le profil.",
              confirmLabel: "Supprimer",
              destructive: true
            });
            if (ok) {
              hide.mutate();
            }
          }}
        />
      )
    });
  }, [confirm, hide, matchId, navigation, other]);

  // Leave when the match disappears (unmatch or block from the other side).
  useEffect(() => {
    if (removedMatchIds.has(matchId)) {
      toast("Cette conversation n'est plus disponible.");
      navigation.goBack();
    }
  }, [matchId, navigation, removedMatchIds, toast]);

  const pages = messages.data?.pages;
  const ordered = useMemo(() => {
    const list: Message[] = [];
    // Pages are newest-first; messages are chronological inside each page.
    (pages ?? [])
      .slice()
      .reverse()
      .forEach((p) => list.push(...p.messages));
    return list;
  }, [pages]);
  const otherLastReadAt = pages?.[0]?.otherLastReadAt ?? null;
  const lastIncoming = [...ordered].reverse().find((m) => m.senderId !== myId)?.id;

  useEffect(() => {
    if (lastIncoming || ordered.length === 0) {
      markRead.mutate();
    }
    // Re-mark read whenever a new incoming message arrives while the thread is open.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastIncoming]);

  const lastMineRead = useMemo(() => {
    if (!otherLastReadAt) {
      return null;
    }
    const readAt = new Date(otherLastReadAt).getTime();
    return (
      [...ordered].reverse().find((m) => m.senderId === myId && new Date(m.createdAt).getTime() <= readAt)?.id ?? null
    );
  }, [myId, ordered, otherLastReadAt]);

  const submit = () => {
    const body = draft.trim();
    if (!body) {
      return;
    }
    setDraft("");
    send.mutate(body, {
      onError: (e) => {
        setDraft(body);
        toast(errorMessage(e), "error");
      }
    });
  };

  if (messages.isLoading) {
    return <LoadingState />;
  }
  if (messages.isError) {
    return <ErrorState message={errorMessage(messages.error)} onRetry={() => void messages.refetch()} />;
  }

  const inverted = [...ordered].reverse();

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: colors.background }]} testID="chat-screen">
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={90}
      >
        {ordered.length === 0 ? (
          <EmptyState
            icon="chatbubbles-outline"
            title={`Dis bonjour à ${other.firstName}`}
            message="Une question sur son profil ou un point commun est souvent le meilleur début."
          />
        ) : (
          <FlatList
            inverted
            data={inverted}
            keyExtractor={(m) => m.id}
            contentContainerStyle={{ padding: spacing.md, gap: 4 }}
            onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
            onEndReachedThreshold={0.3}
            renderItem={({ item, index }) => {
              const mine = item.senderId === myId;
              const older = inverted[index + 1];
              const showDay = !older || !isSameDay(older.createdAt, item.createdAt);
              const pending = item.id.startsWith("pending-");
              return (
                <View>
                  {showDay ? (
                    <Text variant="caption" tone="subtle" align="center" style={styles.day}>
                      {formatDayHeader(item.createdAt)}
                    </Text>
                  ) : null}
                  <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
                    <View
                      style={[
                        styles.bubble,
                        {
                          borderRadius: radii.lg,
                          backgroundColor: mine ? colors.primary : colors.surface,
                          borderColor: mine ? colors.primary : colors.border,
                          opacity: pending ? 0.6 : 1
                        },
                        mine ? { borderBottomRightRadius: 6 } : { borderBottomLeftRadius: 6 }
                      ]}
                    >
                      <Text style={{ color: mine ? colors.onPrimary : colors.text }} testID="chat-message">
                        {item.body}
                      </Text>
                    </View>
                    <Text variant="caption" tone="subtle" style={styles.meta}>
                      {pending ? "Envoi…" : formatTime(item.createdAt)}
                      {item.id === lastMineRead ? " · Vu" : ""}
                    </Text>
                  </View>
                </View>
              );
            }}
          />
        )}
        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.surface }]}>
          <TextInput
            testID="chat-input"
            accessibilityLabel="Écrire un message"
            placeholder="Ton message…"
            placeholderTextColor={colors.textSubtle}
            value={draft}
            onChangeText={setDraft}
            multiline
            maxLength={2000}
            onSubmitEditing={Platform.OS === "web" ? submit : undefined}
            blurOnSubmit={false}
            onKeyPress={(e) => {
              const ev = e.nativeEvent as { key: string; shiftKey?: boolean };
              if (Platform.OS === "web" && ev.key === "Enter" && !ev.shiftKey) {
                e.preventDefault();
                submit();
              }
            }}
            style={[
              styles.input,
              { color: colors.text, backgroundColor: colors.surfaceSunken, borderRadius: radii.lg }
            ]}
          />
          <IconButton
            testID="chat-send"
            icon="arrow-up"
            label="Envoyer"
            filled
            tone="primary"
            onPress={submit}
            disabled={!draft.trim()}
          />
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  headerTitle: { flexDirection: "row", alignItems: "center", gap: 10 },
  day: { marginVertical: 12, textTransform: "capitalize" },
  bubbleRow: { maxWidth: "80%", gap: 2, marginVertical: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: 14, paddingVertical: 10, borderWidth: 1 },
  meta: { paddingHorizontal: 6 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 8,
    padding: 10,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: { flex: 1, minHeight: 44, maxHeight: 140, paddingHorizontal: 16, paddingVertical: 11, fontSize: 16 }
});
