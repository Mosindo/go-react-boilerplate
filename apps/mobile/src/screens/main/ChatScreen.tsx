import { Ionicons } from "@expo/vector-icons";
import { useIsFocused } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { fetchMessages, hideConversation, markConversationRead, sendMessage } from "../../api/chat";
import { unmatchConversation } from "../../api/discovery";
import { ApiError } from "../../api/client";
import type { Message } from "../../api/types";
import { Button } from "../../components/Button";
import { SafetySheet } from "../../components/SafetySheet";
import { Sheet } from "../../components/Sheet";
import { EmptyView, ErrorView, LoadingView } from "../../components/StateViews";
import { queryKeys } from "../../hooks/queries";
import { errorMessage, useAsyncAction } from "../../hooks/useAsyncAction";
import { formatClock, formatDayLabel, isSameDay } from "../../lib/format";
import type { RootStackParamList } from "../../navigation/types";
import { useMe } from "../../state/AuthProvider";
import { useRealtime } from "../../state/RealtimeProvider";
import { appendMessage, flattenMessages, type MessagesCache } from "../../state/messageCache";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;

const MAX_LENGTH = 2000;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId, user } = route.params;
  const theme = useTheme();
  const toast = useToast();
  const me = useMe();
  const queryClient = useQueryClient();
  const focused = useIsFocused();
  const { connected } = useRealtime();
  const [text, setText] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [safetyOpen, setSafetyOpen] = useState(false);
  const markingRead = useRef(false);

  const query = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => last.nextBefore ?? undefined,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 2
  });

  const chronological = useMemo(() => flattenMessages(query.data), [query.data]);
  const newestFirst = useMemo(() => [...chronological].reverse(), [chronological]);
  const lastSeenId = useMemo(
    () => chronological.filter((m) => m.senderId === me.id && m.readAt !== null).at(-1)?.id,
    [chronological, me.id]
  );
  const gone = query.error instanceof ApiError && query.error.status === 404;

  const close = useCallback(() => navigation.goBack(), [navigation]);

  // Open the profile from the header.
  useEffect(() => {
    navigation.setOptions({
      title: user.firstName,
      headerRight: () => (
        <Pressable
          accessibilityLabel="Options de la conversation"
          accessibilityRole="button"
          hitSlop={12}
          onPress={() => setMenuOpen(true)}
        >
          <Ionicons color={theme.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      )
    });
  }, [navigation, user.firstName, theme.text]);

  // Mark incoming messages as read while the conversation is on screen.
  const hasUnread = chronological.some((m) => m.senderId !== me.id && m.readAt === null);
  useEffect(() => {
    if (!focused || !hasUnread || markingRead.current) return;
    markingRead.current = true;
    markConversationRead(conversationId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      })
      .catch(() => undefined)
      .finally(() => {
        markingRead.current = false;
      });
  }, [focused, hasUnread, conversationId, queryClient, chronological.length]);

  const send = useAsyncAction(async () => {
    const body = text.trim();
    if (!body) return;
    setText("");
    try {
      const message = await sendMessage(conversationId, body);
      queryClient.setQueryData<MessagesCache>(queryKeys.messages(conversationId), (cache) =>
        appendMessage(cache, message)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    } catch (error) {
      setText(body); // never lose what the user typed
      throw error;
    }
  });

  const hide = useAsyncAction(async () => {
    await hideConversation(conversationId);
    queryClient.removeQueries({ queryKey: queryKeys.messages(conversationId) });
    await queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    close();
  });

  const doUnmatch = useAsyncAction(async () => {
    await unmatchConversation(conversationId);
    queryClient.removeQueries({ queryKey: queryKeys.messages(conversationId) });
    await queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    close();
  });

  useEffect(() => {
    if (send.error) toast.show(send.error, "error");
  }, [send.error, toast]);

  const renderItem = ({ item, index }: { item: Message; index: number }) => {
    const mine = item.senderId === me.id;
    const older = newestFirst[index + 1];
    const showDay = !older || !isSameDay(older.createdAt, item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text style={[styles.day, { color: theme.textMuted }]}>
            {formatDayLabel(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
          <View
            style={[
              styles.bubble,
              { backgroundColor: mine ? theme.bubbleMine : theme.bubbleTheirs },
              mine ? styles.bubbleMine : styles.bubbleTheirs
            ]}
          >
            <Text style={[styles.body, { color: mine ? "#FFFFFF" : theme.text }]}>{item.body}</Text>
            <Text
              style={[styles.time, { color: mine ? "rgba(255,255,255,0.8)" : theme.textMuted }]}
            >
              {formatClock(item.createdAt)}
            </Text>
          </View>
        </View>
        {item.id === lastSeenId ? (
          <Text style={[styles.seen, { color: theme.textMuted }]}>Vu</Text>
        ) : null}
      </View>
    );
  };

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
        style={styles.flex}
      >
        {!connected ? (
          <Text
            style={[
              styles.offline,
              { backgroundColor: theme.surfaceMuted, color: theme.textMuted }
            ]}
          >
            Reconnexion en cours…
          </Text>
        ) : null}

        {query.isLoading ? (
          <LoadingView />
        ) : gone ? (
          <EmptyView
            actionLabel="Retour"
            message="Elle a peut-être été supprimée ou le match a pris fin."
            onAction={close}
            title="Cette conversation n'existe plus"
          />
        ) : query.isError ? (
          <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
        ) : (
          <FlatList
            contentContainerStyle={styles.list}
            data={newestFirst}
            inverted
            keyExtractor={(m) => String(m.id)}
            ListEmptyComponent={
              <Text style={[styles.empty, { color: theme.textMuted }]}>
                Vous avez matché avec {user.firstName}. Dites bonjour !
              </Text>
            }
            onEndReached={() => {
              if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
            }}
            onEndReachedThreshold={0.4}
            renderItem={renderItem}
            style={styles.flex}
          />
        )}

        {!gone ? (
          <View
            style={[
              styles.composer,
              { borderTopColor: theme.border, backgroundColor: theme.background }
            ]}
          >
            <TextInput
              accessibilityLabel="Votre message"
              maxLength={MAX_LENGTH}
              multiline
              onChangeText={setText}
              placeholder="Votre message…"
              placeholderTextColor={theme.textMuted}
              style={[
                styles.input,
                { backgroundColor: theme.surface, borderColor: theme.border, color: theme.text }
              ]}
              testID="chat-input"
              value={text}
            />
            <Pressable
              accessibilityLabel="Envoyer"
              accessibilityRole="button"
              accessibilityState={{ disabled: !text.trim() }}
              disabled={!text.trim() || send.loading}
              onPress={() => void send.run()}
              style={[
                styles.send,
                { backgroundColor: theme.primary, opacity: text.trim() ? 1 : 0.4 }
              ]}
              testID="chat-send"
            >
              <Ionicons color="#FFFFFF" name="arrow-up" size={22} />
            </Pressable>
          </View>
        ) : null}
      </KeyboardAvoidingView>

      <Sheet onClose={() => setMenuOpen(false)} title={user.firstName} visible={menuOpen}>
        <View style={styles.menu}>
          <Button
            onPress={() => {
              setMenuOpen(false);
              navigation.navigate("ProfileDetail", { userId: user.id, firstName: user.firstName });
            }}
            title="Voir le profil"
            variant="secondary"
          />
          <Button
            onPress={() => {
              setMenuOpen(false);
              void hide.run();
            }}
            title="Supprimer la conversation"
            variant="secondary"
          />
          <Button
            onPress={() => {
              setMenuOpen(false);
              Alert.alert(
                `Annuler le match avec ${user.firstName} ?`,
                "La conversation sera supprimée pour vous deux.",
                [
                  { text: "Annuler", style: "cancel" },
                  { text: "Confirmer", style: "destructive", onPress: () => void doUnmatch.run() }
                ]
              );
            }}
            title="Annuler le match"
            variant="secondary"
          />
          <Button
            onPress={() => {
              setMenuOpen(false);
              setSafetyOpen(true);
            }}
            title="Signaler ou bloquer"
            variant="danger"
          />
        </View>
      </Sheet>

      <SafetySheet
        firstName={user.firstName}
        onBlocked={() => {
          setSafetyOpen(false);
          close();
        }}
        onClose={() => setSafetyOpen(false)}
        userId={user.id}
        visible={safetyOpen}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.md },
  offline: { textAlign: "center", fontSize: fontSizes.xs, paddingVertical: 4 },
  day: {
    textAlign: "center",
    fontSize: fontSizes.xs,
    fontWeight: "600",
    marginVertical: spacing.md
  },
  bubbleRow: { flexDirection: "row", marginVertical: 2 },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: {
    maxWidth: "80%",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg
  },
  bubbleMine: { borderBottomRightRadius: 6 },
  bubbleTheirs: { borderBottomLeftRadius: 6 },
  body: { fontSize: fontSizes.md, lineHeight: 22 },
  time: { fontSize: 11, marginTop: 2, alignSelf: "flex-end" },
  seen: { fontSize: 11, textAlign: "right", marginRight: spacing.sm, marginTop: 2 },
  empty: { textAlign: "center", padding: spacing.xl, transform: [{ scaleY: -1 }] },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    gap: spacing.sm
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderWidth: 1,
    borderRadius: 22,
    paddingHorizontal: spacing.lg,
    paddingTop: 12,
    paddingBottom: 12,
    fontSize: fontSizes.md
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  menu: { gap: spacing.sm }
});
