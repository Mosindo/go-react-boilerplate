import { Ionicons } from "@expo/vector-icons";
import { useIsFocused, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FlatList, Pressable, TextInput, View } from "react-native";
import { ApiError } from "../api/client";
import { blockUser, clearConversation, markConversationRead, sendMessage, unmatch } from "../api/platform";
import type { Message } from "../api/types";
import { ActionMenu, type MenuAction } from "../components/ActionMenu";
import { ReportSheet } from "../components/ReportSheet";
import { useCurrentUser } from "../hooks/useAuth";
import { useMessages } from "../hooks/queries";
import { confirm } from "../lib/confirm";
import { visiblePending, type PendingMessage } from "../lib/chat";
import { formatDayLabel, isSameDay } from "../lib/dates";
import { errorMessage } from "../lib/errors";
import type { RootScreenProps, RootStackParamList } from "../navigation/types";
import { appendMessage, queryKeys, type MessagesData } from "../realtime/cache";
import { EmptyView, ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Avatar, IconButton, MessageItem, Text } from "../shared/ui";
import { SafeAreaLayout } from "../shared/layout";
import { KeyboardAvoidingView, Platform } from "react-native";
import { layout, radii, spacing, useStyles, useTheme, type Colors } from "../theme";

type Row = { key: string; message: Message | PendingMessage; pending: boolean; showDay: boolean };

const MAX_LENGTH = 2000;

export default function ChatScreen({ route }: RootScreenProps<"Chat">) {
  const { conversationId, matchId, userId, name, photoPath } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const me = useCurrentUser();
  const focused = useIsFocused();
  const { colors } = useTheme();
  const styles = useStyles(makeStyles);
  const query = useMessages(conversationId);
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [text, setText] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const [reporting, setReporting] = useState(false);
  const inputRef = useRef<TextInput>(null);

  // Chronological list (oldest → newest): pages are stored newest page first.
  const messages = useMemo(() => [...(query.data?.pages ?? [])].reverse().flatMap((page) => page.messages), [query.data]);

  const rows = useMemo<Row[]>(() => {
    const all: { id: string; message: Message | PendingMessage; pending: boolean }[] = [
      ...messages.map((m) => ({ id: m.id, message: m, pending: false })),
      ...visiblePending(messages, pending, me.id).map((p) => ({ id: p.tempId, message: p, pending: true }))
    ];
    // Newest first because the list is inverted; a day label goes above the first message of each day.
    return all
      .map((item, i) => ({
        key: item.id,
        message: item.message,
        pending: item.pending,
        showDay: i === 0 || !isSameDay(all[i - 1].message.createdAt, item.message.createdAt)
      }))
      .reverse();
  }, [messages, pending, me.id]);

  const unreadFromOther = messages.some((m) => m.senderUserId !== me.id && !m.readAt);

  // Mark the conversation as read while it is on screen.
  useEffect(() => {
    if (!focused || !unreadFromOther) return;
    void markConversationRead(conversationId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      })
      .catch(() => undefined);
  }, [focused, unreadFromOther, conversationId, queryClient]);

  const send = useCallback(
    async (content: string, existingTempId?: string) => {
      const tempId = existingTempId ?? `pending-${Date.now()}-${Math.random().toString(36).slice(2)}`;
      setPending((list) =>
        existingTempId
          ? list.map((p) => (p.tempId === tempId ? { ...p, failed: false } : p))
          : [...list, { tempId, content, createdAt: new Date().toISOString(), failed: false }]
      );
      try {
        const sent = await sendMessage(conversationId, content);
        queryClient.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) => appendMessage(data, sent));
        setPending((list) => list.filter((p) => p.tempId !== tempId));
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) {
          setPending((list) => list.filter((p) => p.tempId !== tempId));
          showError("Cette conversation n'est plus disponible.");
          navigation.goBack();
          return;
        }
        setPending((list) => list.map((p) => (p.tempId === tempId ? { ...p, failed: true } : p)));
        showError(errorMessage(e));
      }
    },
    [conversationId, navigation, queryClient]
  );

  const submit = () => {
    const content = text.trim();
    if (!content) return;
    setText("");
    void send(content);
    inputRef.current?.focus();
  };

  const leave = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    navigation.goBack();
  };

  const actions: MenuAction[] = [
    {
      key: "profile",
      label: `Voir le profil de ${name}`,
      icon: "person-outline",
      onPress: () => navigation.navigate("ProfileDetail", { userId, matchId, conversationId })
    },
    {
      key: "clear",
      label: "Effacer la conversation pour moi",
      icon: "trash-outline",
      onPress: async () => {
        if (!(await confirm("Effacer la conversation ?", `L'historique disparaît de votre côté. ${name} le conserve.`, "Effacer", true)))
          return;
        try {
          await clearConversation(conversationId);
          showSuccess("Conversation effacée.");
          leave();
        } catch (e) {
          showError(errorMessage(e));
        }
      }
    },
    {
      key: "unmatch",
      label: "Annuler le match",
      icon: "heart-dislike-outline",
      destructive: true,
      onPress: async () => {
        if (
          !(await confirm(
            "Annuler le match ?",
            `Vous ne pourrez plus discuter avec ${name}, et vous ne vous reverrez pas.`,
            "Annuler le match",
            true
          ))
        )
          return;
        try {
          await unmatch(matchId);
          leave();
        } catch (e) {
          showError(errorMessage(e));
        }
      }
    },
    { key: "report", label: "Signaler", icon: "flag-outline", destructive: true, onPress: () => setReporting(true) },
    {
      key: "block",
      label: "Bloquer",
      icon: "ban-outline",
      destructive: true,
      onPress: async () => {
        if (!(await confirm(`Bloquer ${name} ?`, "Vous ne vous verrez plus et la conversation disparaîtra.", "Bloquer", true))) return;
        try {
          await blockUser(userId);
          showSuccess(`${name} est bloqué(e).`);
          leave();
        } catch (e) {
          showError(errorMessage(e));
        }
      }
    }
  ];

  return (
    <SafeAreaLayout edges={["top", "bottom"]}>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <View style={styles.page}>
          <View style={styles.header}>
            <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} testID="chat-back" />
            <Pressable
              style={styles.who}
              accessibilityRole="button"
              accessibilityLabel={`Profil de ${name}`}
              onPress={() => navigation.navigate("ProfileDetail", { userId, matchId, conversationId })}
            >
              <Avatar name={name} path={photoPath} size={40} />
              <Text variant="heading" numberOfLines={1} style={styles.name}>
                {name}
              </Text>
            </Pressable>
            <IconButton testID="chat-menu" icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenuOpen(true)} />
          </View>

          {query.isLoading ? <LoadingView /> : null}
          {query.isError ? <ErrorView error={query.error} onRetry={() => void query.refetch()} /> : null}
          {query.data && rows.length === 0 ? (
            <EmptyView
              icon="chatbubbles-outline"
              title={`Dites bonjour à ${name} !`}
              message="Un premier message sympa vaut mieux qu'un simple « salut »."
            />
          ) : null}
          {query.data && rows.length > 0 ? (
            <FlatList
              testID="chat-list"
              data={rows}
              inverted
              keyExtractor={(row) => row.key}
              onEndReached={() => {
                if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
              }}
              onEndReachedThreshold={0.4}
              ListFooterComponent={query.isFetchingNextPage ? <LoadingView /> : null}
              renderItem={({ item }) => {
                const m = item.message;
                const isPending = item.pending;
                return (
                  <View>
                    {item.showDay ? (
                      <Text variant="caption" tone="muted" center style={styles.day}>
                        {formatDayLabel(m.createdAt)}
                      </Text>
                    ) : null}
                    <Pressable
                      disabled={!isPending || !(m as PendingMessage).failed}
                      onPress={() => void send(m.content, (m as PendingMessage).tempId)}
                      accessibilityHint={isPending && (m as PendingMessage).failed ? "Appuyez pour renvoyer" : undefined}
                    >
                      <MessageItem
                        content={m.content}
                        createdAt={m.createdAt}
                        mine={isPending || (m as Message).senderUserId === me.id}
                        read={!isPending && !!(m as Message).readAt}
                        failed={isPending && (m as PendingMessage).failed}
                      />
                    </Pressable>
                  </View>
                );
              }}
            />
          ) : (
            <View style={styles.fill} />
          )}

          <View style={styles.composer}>
            <TextInput
              ref={inputRef}
              testID="chat-input"
              accessibilityLabel="Votre message"
              value={text}
              onChangeText={setText}
              placeholder="Votre message…"
              placeholderTextColor={colors.textMuted}
              multiline
              maxLength={MAX_LENGTH}
              style={styles.input}
              onKeyPress={(e) => {
                // Web: Enter sends, Shift+Enter inserts a new line.
                const native = e.nativeEvent as unknown as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  e.preventDefault();
                  submit();
                }
              }}
            />
            <Pressable
              testID="chat-send"
              accessibilityRole="button"
              accessibilityLabel="Envoyer"
              disabled={!text.trim()}
              onPress={submit}
              style={[styles.send, !text.trim() && styles.sendDisabled]}
            >
              <Ionicons name="arrow-up" size={22} color={colors.onPrimary} />
            </Pressable>
          </View>
        </View>
      </KeyboardAvoidingView>

      <ActionMenu visible={menuOpen} actions={actions} onClose={() => setMenuOpen(false)} />
      <ReportSheet
        visible={reporting}
        userId={userId}
        name={name}
        onClose={() => setReporting(false)}
        onDone={(blocked) => blocked && leave()}
      />
    </SafeAreaLayout>
  );
}

const makeStyles = (colors: Colors) => ({
  fill: { flex: 1 },
  page: { flex: 1, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" } as const,
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.xs,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    minHeight: 56
  } as const,
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm } as const,
  name: { flex: 1 },
  day: { marginVertical: spacing.md },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  } as const,
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderRadius: radii.xl,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    color: colors.text,
    paddingHorizontal: spacing.lg,
    paddingTop: 11,
    paddingBottom: 11,
    fontSize: 16
  },
  send: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  } as const,
  sendDisabled: { opacity: 0.4 }
});
