import { Ionicons } from "@expo/vector-icons";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { errorMessage, isApiError } from "../api/client";
import { datingApi } from "../api/dating";
import type { ChatMessage, MessagePage } from "../api/types";
import { ActionMenu } from "../components/ActionMenu";
import { keys } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { appendMessage } from "../hooks/useRealtime";
import { formatDayLabel, formatTime, sameDay } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { radii, spacing, useTheme } from "../theme";
import { EmptyState, ErrorState, LoadingState, Text, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

export default function ChatScreen({ navigation, route }: Props) {
  const { matchId, userId, name } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const client = useQueryClient();
  const insets = useSafeAreaInsets();
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [menu, setMenu] = useState(false);
  const lastMarked = useRef<string | null>(null);

  const query = useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => datingApi.messages(matchId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: MessagePage) => last.nextBefore || undefined,
    staleTime: 0
  });

  const messages: ChatMessage[] = query.data?.pages.flatMap((p) => p.items) ?? [];
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  const newestId = messages[0]?.id ?? null;
  const newestFromOther = messages[0] ? messages[0].senderId !== user?.id : false;

  // Mark the conversation as read whenever it is open and a new message from the other side shows up.
  useEffect(() => {
    if (!query.isSuccess) {
      return;
    }
    const marker = newestId ?? "none";
    if (lastMarked.current === marker) {
      return;
    }
    const first = lastMarked.current === null;
    lastMarked.current = marker;
    if (!first && !newestFromOther) {
      return; // our own message: nothing new to acknowledge
    }
    datingApi
      .markRead(matchId)
      .then(() => client.invalidateQueries({ queryKey: keys.matches }))
      .catch(() => undefined);
  }, [client, matchId, newestFromOther, newestId, query.isSuccess]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerStyle: { backgroundColor: colors.background },
      headerTintColor: colors.text,
      headerShadowVisible: false,
      headerRight: () => (
        <Pressable accessibilityRole="button" accessibilityLabel="Options de la conversation" onPress={() => setMenu(true)} hitSlop={10}>
          <Ionicons name="ellipsis-horizontal" size={24} color={colors.text} />
        </Pressable>
      )
    });
  }, [colors, name, navigation]);

  const send = useCallback(async () => {
    const body = text.trim();
    if (!body || sending) {
      return;
    }
    setSending(true);
    try {
      const message = await datingApi.sendMessage(matchId, body);
      client.setQueryData(keys.messages(matchId), (cache: Parameters<typeof appendMessage>[0]) => appendMessage(cache, message));
      void client.invalidateQueries({ queryKey: keys.matches });
      setText("");
    } catch (error) {
      if (isApiError(error, 404)) {
        toast.show("Cette conversation n'existe plus.", "error");
        navigation.goBack();
        return;
      }
      toast.show(isApiError(error, 429) ? "Doucement, vous envoyez trop de messages." : errorMessage(error), "error");
    } finally {
      setSending(false);
    }
  }, [client, matchId, navigation, sending, text, toast]);

  const run = async (task: () => Promise<void>, done: string, leave = false) => {
    try {
      await task();
      toast.show(done, "success");
      await client.invalidateQueries({ queryKey: keys.matches });
      if (leave) {
        navigation.goBack();
      } else {
        await client.invalidateQueries({ queryKey: keys.messages(matchId) });
      }
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  let lastOwnId: string | null = null;
  for (const m of messages) {
    if (m.senderId === user?.id) {
      lastOwnId = m.id;
      break;
    }
  }

  const renderItem = ({ item, index }: { item: ChatMessage; index: number }) => {
    const mine = item.senderId === user?.id;
    const older = messages[index + 1];
    const showDay = !older || !sameDay(new Date(older.createdAt), new Date(item.createdAt));
    const read = mine && item.id === lastOwnId && otherLastReadAt !== null && new Date(item.createdAt) <= new Date(otherLastReadAt);
    return (
      <View>
        {showDay ? (
          <Text variant="caption" tone="muted" style={styles.day}>
            {formatDayLabel(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
          <View
            style={[
              styles.bubble,
              { backgroundColor: mine ? colors.primary : colors.surface, borderColor: colors.border, borderWidth: mine ? 0 : 1 }
            ]}
          >
            <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.body}</Text>
            <Text variant="caption" style={{ color: mine ? colors.onPrimary : colors.textMuted, opacity: 0.8, alignSelf: "flex-end" }}>
              {formatTime(item.createdAt)}
            </Text>
          </View>
        </View>
        {item.id === lastOwnId ? (
          <Text variant="caption" tone="muted" style={styles.receipt}>
            {read ? "Lu" : "Envoyé"}
          </Text>
        ) : null}
      </View>
    );
  };

  let body: React.ReactNode;
  if (query.isLoading) {
    body = <LoadingState />;
  } else if (query.isError) {
    body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  } else if (messages.length === 0) {
    body = (
      <EmptyState
        icon="chatbubble-ellipses-outline"
        title={`Dites bonjour à ${name}`}
        message="Un message simple et sincère est souvent le meilleur début."
      />
    );
  } else {
    body = (
      <FlatList
        inverted
        data={messages}
        keyExtractor={(m) => m.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        keyboardShouldPersistTaps="handled"
      />
    );
  }

  return (
    <KeyboardAvoidingView
      style={[styles.flex, { backgroundColor: colors.background }]}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={90}
    >
      <View style={styles.flex}>{body}</View>
      <View
        style={[
          styles.composer,
          { borderTopColor: colors.border, backgroundColor: colors.surface, paddingBottom: Math.max(insets.bottom, spacing.sm) }
        ]}
      >
        <TextInput
          accessibilityLabel="Votre message"
          value={text}
          onChangeText={setText}
          placeholder="Écrire un message…"
          placeholderTextColor={colors.textMuted}
          multiline
          maxLength={2000}
          style={[styles.input, { color: colors.text, backgroundColor: colors.surfaceMuted }]}
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Envoyer"
          disabled={!text.trim() || sending}
          onPress={() => void send()}
          style={[styles.send, { backgroundColor: colors.primary, opacity: !text.trim() || sending ? 0.45 : 1 }]}
        >
          <Ionicons name="arrow-up" size={22} color={colors.onPrimary} />
        </Pressable>
      </View>
      <ActionMenu
        visible={menu}
        title={name}
        onClose={() => setMenu(false)}
        actions={[
          { label: "Voir le profil", onPress: () => navigation.navigate("UserProfile", { userId, matchId }) },
          {
            label: "Effacer la conversation pour moi",
            onPress: () => void run(() => datingApi.clearConversation(matchId), "Conversation effacée")
          },
          {
            label: "Se désassocier (unmatch)",
            destructive: true,
            onPress: () => void run(() => datingApi.unmatch(matchId), "Match supprimé", true)
          },
          { label: "Signaler", destructive: true, onPress: () => navigation.navigate("Report", { userId, name }) },
          { label: "Bloquer", destructive: true, onPress: () => void run(() => datingApi.block(userId), `${name} est bloqué·e`, true) }
        ]}
      />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.lg, gap: spacing.xs },
  day: { textAlign: "center", marginVertical: spacing.md },
  bubbleRow: { flexDirection: "row", marginVertical: 2 },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, gap: 2 },
  receipt: { textAlign: "right", marginTop: 2 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingTop: 12,
    paddingBottom: 12,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
