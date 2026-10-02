import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { matchesApi } from "../../api/endpoints";
import type { Message, MessagePage } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { formatClock } from "../../lib/format";
import { useSafetyActions } from "../../lib/useSafetyActions";
import type { MainStackParamList } from "../../navigation/types";
import { keys, mergeMessage } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { useFeedback } from "../../ui/Feedback";
import { ErrorState, Loading, errorMessage } from "../../ui/States";
import { Text } from "../../ui/Text";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

export function ChatScreen({ navigation, route }: Props) {
  const { colors } = useTheme();
  const { user } = useAuth();
  const qc = useQueryClient();
  const { toast, sheet, confirm } = useFeedback();
  const { block, report } = useSafetyActions();
  const { matchId, name, userId } = route.params;
  const myId = user?.id ?? "";

  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const listRef = useRef<FlatList<Message>>(null);

  const query = useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => matchesApi.messages(matchId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[0]?.id : undefined),
  });

  // Newest first, as the list is inverted.
  const messages = useMemo(
    () =>
      query.data
        ? [...query.data.pages].flatMap((p) => p.messages).sort((a, b) => b.id - a.id)
        : [],
    [query.data],
  );
  const lastIncomingId = messages.find((m) => m.senderId !== myId)?.id;
  const lastMineId = messages.find((m) => m.senderId === myId)?.id;

  // Mark as read on open and whenever the other person writes while the screen is open.
  useEffect(() => {
    if (!query.data) return;
    void matchesApi
      .markRead(matchId)
      .then(() => {
        void qc.invalidateQueries({ queryKey: keys.matches });
        void qc.invalidateQueries({ queryKey: keys.summary });
        void qc.invalidateQueries({ queryKey: keys.notifications });
      })
      .catch(() => undefined);
  }, [matchId, lastIncomingId, query.data, qc]);

  const leave = useCallback(() => navigation.popToTop(), [navigation]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          hitSlop={12}
          onPress={openMenu}
        >
          <Ionicons name="ellipsis-horizontal" size={24} color={colors.text} />
        </Pressable>
      ),
    });
    // openMenu only closes over stable values listed below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigation, name, colors.text]);

  function openMenu() {
    sheet(name, [
      { label: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId }) },
      {
        label: "Supprimer l'historique pour moi",
        onPress: async () => {
          if (
            !(await confirm({
              title: "Supprimer l'historique ?",
              message: "Les messages disparaissent pour vous uniquement.",
              confirmLabel: "Supprimer",
              destructive: true,
            }))
          )
            return;
          try {
            await matchesApi.clear(matchId);
            qc.setQueryData<InfiniteData<MessagePage>>(keys.messages(matchId), (d) =>
              d ? { ...d, pages: d.pages.map((p) => ({ ...p, messages: [], hasMore: false })) } : d,
            );
            void qc.invalidateQueries({ queryKey: keys.matches });
          } catch (err) {
            toast(errorMessage(err), "error");
          }
        },
      },
      {
        label: "Annuler le match",
        destructive: true,
        onPress: async () => {
          if (
            !(await confirm({
              title: `Annuler le match avec ${name} ?`,
              message: "La conversation sera supprimée pour vous deux.",
              confirmLabel: "Annuler le match",
              destructive: true,
            }))
          )
            return;
          try {
            await matchesApi.unmatch(matchId);
            void qc.invalidateQueries({ queryKey: keys.matches });
            leave();
          } catch (err) {
            toast(errorMessage(err), "error");
          }
        },
      },
      { label: "Signaler", onPress: () => report(userId, name, leave) },
      { label: "Bloquer", destructive: true, onPress: () => void block(userId, name, leave) },
    ]);
  }

  const send = async () => {
    const body = text.trim();
    if (!body || sending) return;
    setSending(true);
    setText("");
    try {
      const message = await matchesApi.send(matchId, body);
      qc.setQueryData<InfiniteData<MessagePage>>(keys.messages(matchId), (d) =>
        mergeMessage(d, message),
      );
      void qc.invalidateQueries({ queryKey: keys.matches });
      listRef.current?.scrollToOffset({ offset: 0, animated: true });
    } catch (err) {
      setText(body); // never lose what the person typed
      toast(errorMessage(err), "error");
    } finally {
      setSending(false);
    }
  };

  if (query.isLoading) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        <FlatList
          ref={listRef}
          testID="chat-list"
          inverted
          data={messages}
          keyExtractor={(m) => String(m.id)}
          contentContainerStyle={styles.list}
          onEndReached={() =>
            query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()
          }
          onEndReachedThreshold={0.3}
          ListEmptyComponent={
            <Text tone="muted" center style={styles.empty}>
              C&apos;est un match ! Écrivez le premier message.
            </Text>
          }
          renderItem={({ item }) => {
            const mine = item.senderId === myId;
            return (
              <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
                <View
                  style={[
                    styles.bubble,
                    {
                      backgroundColor: mine ? colors.primary : colors.surface,
                      borderColor: colors.border,
                    },
                    mine && { borderBottomRightRadius: 4 },
                    !mine && { borderBottomLeftRadius: 4, borderWidth: 1 },
                  ]}
                >
                  <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.body}</Text>
                  <Text
                    variant="caption"
                    style={{
                      color: mine ? colors.onPrimary : colors.textMuted,
                      alignSelf: "flex-end",
                      opacity: 0.85,
                    }}
                  >
                    {formatClock(item.createdAt)}
                    {mine && item.id === lastMineId && item.readAt ? " · Lu" : ""}
                  </Text>
                </View>
              </View>
            );
          }}
        />
        <View
          style={[
            styles.composer,
            { borderTopColor: colors.border, backgroundColor: colors.surface },
          ]}
        >
          <TextInput
            testID="chat-input"
            accessibilityLabel="Votre message"
            value={text}
            onChangeText={setText}
            placeholder="Votre message…"
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={2000}
            style={[styles.input, { color: colors.text, backgroundColor: colors.surfaceAlt }]}
          />
          <Pressable
            testID="chat-send"
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            disabled={!text.trim() || sending}
            onPress={() => void send()}
            style={[
              styles.send,
              { backgroundColor: colors.primary, opacity: !text.trim() || sending ? 0.45 : 1 },
            ]}
          >
            <Ionicons name="arrow-up" size={22} color={colors.onPrimary} />
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.lg, gap: spacing.sm },
  empty: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  bubbleRow: { flexDirection: "row" },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: {
    maxWidth: "80%",
    borderRadius: radii.lg,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    gap: 2,
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingTop: 12,
    paddingBottom: 12,
    fontSize: 16,
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
});
