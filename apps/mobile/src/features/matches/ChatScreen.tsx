import React, { useEffect, useMemo, useRef, useState } from "react";
import { FlatList, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useIsFocused } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../../api/client";
import { matchesApi } from "../../api/endpoints";
import type { Message } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { keys, useMessages, useSendMessage } from "../../hooks/queries";
import { useRealtimeConnected } from "../../hooks/useRealtime";
import type { MainScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { formatDayHeader, formatTime } from "../../utils/format";

const MAX_LENGTH = 2000;

export function ChatScreen({ route, navigation }: MainScreenProps<"Chat">) {
  const { matchId } = route.params;
  const theme = useTheme();
  const qc = useQueryClient();
  const { user } = useAuth();
  const focused = useIsFocused();
  const online = useRealtimeConnected();
  const match = useQuery({ queryKey: ["match", matchId], queryFn: () => matchesApi.get(matchId) });
  const messages = useMessages(matchId, !online);
  const send = useSendMessage(matchId);
  const [draft, setDraft] = useState("");
  const input = useRef<TextInput>(null);

  const items = useMemo(() => messages.data?.pages.flatMap((p) => p.items) ?? [], [messages.data]);
  const latestIncoming = items.find((m) => m.senderId !== user?.id);
  const lastMineReadId = items.find((m) => m.senderId === user?.id && m.readAt)?.id;

  // Mark the conversation read when it is opened and whenever a new incoming
  // message arrives while the screen is visible.
  useEffect(() => {
    if (!focused || !latestIncoming || latestIncoming.readAt) return;
    matchesApi
      .markRead(matchId)
      .then(() => {
        void qc.invalidateQueries({ queryKey: keys.matches });
        void qc.invalidateQueries({ queryKey: keys.notifications });
      })
      .catch(() => undefined);
  }, [focused, latestIncoming, matchId, qc]);

  const gone = (messages.error instanceof ApiError && messages.error.status === 404) || (match.error instanceof ApiError && match.error.status === 404);
  useEffect(() => {
    if (gone) showToast("Cette conversation n'existe plus.", "info");
  }, [gone]);

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body, {
      onError: (e) => {
        setDraft(body);
        showToast(errorMessage(e), "error");
      }
    });
  };

  const profile = match.data?.profile;
  const name = profile?.firstName ?? route.params.name;
  const header = (
    <View style={[styles.header, { borderColor: theme.colors.border }]}>
      <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} />
      <Pressable
        accessibilityHint="Voir le profil et les options de sécurité"
        accessibilityRole="button"
        disabled={!profile}
        onPress={() => profile && navigation.navigate("ProfileDetail", { userId: profile.userId, matchId })}
        style={styles.headerPerson}
      >
        <Avatar name={name} photo={profile?.photos[0]} size={38} />
        <Text numberOfLines={1} style={styles.flex} variant="heading">
          {name}
        </Text>
      </Pressable>
      {profile ? (
        <IconButton icon="ellipsis-horizontal" label="Options" onPress={() => navigation.navigate("ProfileDetail", { userId: profile.userId, matchId })} />
      ) : null}
    </View>
  );

  let body: React.ReactNode;
  if (gone) {
    body = (
      <EmptyState
        actionLabel="Retour"
        icon="chatbubble-ellipses-outline"
        message="Cette personne n'est plus disponible."
        onAction={() => navigation.goBack()}
        title="Conversation terminée"
      />
    );
  } else if (messages.isLoading) {
    body = <LoadingState />;
  } else if (messages.error) {
    body = <ErrorState onRetry={() => messages.refetch()} />;
  } else {
    body = (
      <FlatList
        ListEmptyComponent={
          <View style={styles.emptyChat}>
            <Text align="center" tone="muted">
              Vous avez matché avec {name}. Dites bonjour 👋
            </Text>
          </View>
        }
        contentContainerStyle={styles.list}
        data={items}
        inverted={items.length > 0}
        keyExtractor={(m) => m.id}
        keyboardShouldPersistTaps="handled"
        onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && messages.fetchNextPage()}
        onEndReachedThreshold={0.3}
        renderItem={({ item, index }) => (
          <Bubble
            message={item}
            mine={item.senderId === user?.id}
            showDay={!items[index + 1] || new Date(items[index + 1].createdAt).toDateString() !== new Date(item.createdAt).toDateString()}
            showRead={item.id === lastMineReadId}
          />
        )}
      />
    );
  }

  return (
    <Screen edges={["top", "left", "right", "bottom"]} padded={false}>
      {header}
      <View style={styles.flex}>{body}</View>
      {!gone ? (
        <View style={[styles.composer, { borderColor: theme.colors.border, backgroundColor: theme.colors.background }]}>
          <TextInput
            ref={input}
            accessibilityLabel="Votre message"
            maxLength={MAX_LENGTH}
            multiline
            numberOfLines={1}
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={theme.colors.textSubtle}
            style={[theme.typography.body, styles.input, { color: theme.colors.text, backgroundColor: theme.colors.surface, borderColor: theme.colors.border }]}
            testID="chat-input"
            value={draft}
          />
          <IconButton
            disabled={!draft.trim() || send.isPending}
            filled
            icon="arrow-up"
            label="Envoyer"
            onPress={submit}
            size={44}
            testID="chat-send"
            tone="primary"
          />
        </View>
      ) : null}
    </Screen>
  );
}

function Bubble({ message, mine, showDay, showRead }: { message: Message; mine: boolean; showDay: boolean; showRead: boolean }) {
  const theme = useTheme();
  return (
    <View>
      {showDay ? (
        <Text align="center" style={styles.day} tone="subtle" variant="caption">
          {formatDayHeader(message.createdAt)}
        </Text>
      ) : null}
      <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
        <View
          style={[
            styles.bubble,
            mine
              ? { backgroundColor: theme.colors.primary, borderBottomRightRadius: 6 }
              : { backgroundColor: theme.colors.surface, borderColor: theme.colors.border, borderWidth: 1, borderBottomLeftRadius: 6 }
          ]}
        >
          <Text tone={mine ? "onPrimary" : "default"}>{message.body}</Text>
        </View>
        <Text style={styles.meta} tone="subtle" variant="caption">
          {formatTime(message.createdAt)}
          {mine ? (showRead ? " · Lu" : message.readAt ? "" : " · Envoyé") : ""}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", gap: 6, paddingHorizontal: 8, paddingVertical: 6, borderBottomWidth: StyleSheet.hairlineWidth },
  headerPerson: { flex: 1, flexDirection: "row", alignItems: "center", gap: 10 },
  list: { paddingHorizontal: 16, paddingVertical: 12, flexGrow: 1 },
  emptyChat: { flex: 1, alignItems: "center", justifyContent: "center", padding: 32 },
  day: { marginVertical: 12, textTransform: "capitalize" },
  bubbleRow: { marginVertical: 3, maxWidth: "80%" },
  left: { alignSelf: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { borderRadius: 20, paddingHorizontal: 14, paddingVertical: 9 },
  meta: { marginTop: 2, marginHorizontal: 4 },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: 10, paddingHorizontal: 12, paddingVertical: 10, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, maxHeight: 120, minHeight: 44, borderWidth: 1, borderRadius: 22, paddingHorizontal: 16, paddingTop: 10, paddingBottom: 10 }
});
