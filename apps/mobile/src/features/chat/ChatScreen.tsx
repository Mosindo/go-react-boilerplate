import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { useIsFocused } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { chatApi, queryKeys, type ConversationItem, type MatchItem } from "../../api";
import { MAX_MESSAGE_LENGTH } from "../../config";
import { findConversation, findMatchByConversation, removeConversation, type Paged } from "../../realtime/cache";
import { setActiveConversation } from "../../realtime/activeConversation";
import { showToast } from "../../shared/feedback";
import { flattenPages } from "../../shared/pagination";
import { ActionSheet, Avatar, EmptyState, ErrorState, IconButton, Text, type ActionSheetOption } from "../../shared/ui";
import { MIN_TOUCH_TARGET, radius, spacing, useTheme } from "../../theme";
import type { MainStackParamList } from "../../navigation/types";
import { useCurrentUser } from "../auth/AuthProvider";
import { usePersonSafety } from "../safety/usePersonSafety";
import { useUnmatch } from "../safety/hooks";
import { useMarkRead, useSendMessage, useThread } from "./hooks";
import { DaySeparator, MessageBubble } from "./MessageBubble";
import { buildThreadRows, type ThreadRow } from "./threadRows";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

const keyExtractor = (row: ThreadRow) => row.key;

function renderRow({ item }: { item: ThreadRow }) {
  return item.kind === "day" ? <DaySeparator label={item.label} /> : <MessageBubble row={item} />;
}

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId, firstName, thumbUrl, matchId: matchIdHint } = route.params;
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const me = useCurrentUser();
  const focused = useIsFocused();
  const thread = useThread(conversationId);
  const [draft, setDraft] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);

  const messages = useMemo(() => flattenPages(thread.data), [thread.data]);
  const rows = useMemo(
    () => buildThreadRows(messages, me.id, Boolean(thread.hasNextPage)),
    [messages, me.id, thread.hasNextPage]
  );

  const leave = useCallback(() => navigation.popToTop(), [navigation]);
  const safety = usePersonSafety({ userId, firstName, conversationId, onBlocked: leave });
  const unmatch = useUnmatch(leave);

  const send = useSendMessage(conversationId, me.id, (body) => {
    setDraft((current) => current || body);
    showToast("Message non envoyé. Réessayez.", "error");
  });
  const markRead = useMarkRead(conversationId, me.id);
  const { mutate: markAsRead, isPending: markingRead } = markRead;

  const hide = useMutation({
    mutationFn: () => chatApi.hide(conversationId),
    onSuccess: () => {
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
        removeConversation(data, conversationId)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
      leave();
    }
  });

  // Track the open thread so socket events read it immediately instead of raising unread counters.
  useEffect(() => {
    if (!focused) {
      return undefined;
    }
    setActiveConversation(conversationId);
    return () => setActiveConversation(null);
  }, [focused, conversationId]);

  const hasUnreadIncoming = messages.some((message) => message.senderId !== me.id && message.readAt === null);
  useEffect(() => {
    if (focused && hasUnreadIncoming && !markingRead) {
      markAsRead();
    }
  }, [focused, hasUnreadIncoming, markingRead, markAsRead]);

  const submit = () => {
    const body = draft.trim();
    if (!body) {
      return;
    }
    setDraft("");
    send.mutate(body);
  };

  const openProfile = () => navigation.navigate("UserProfile", { userId });

  const resolveMatchId = (): string | undefined =>
    matchIdHint ??
    findConversation(queryClient.getQueryData<Paged<ConversationItem>>(queryKeys.conversations), conversationId)
      ?.matchId ??
    findMatchByConversation(queryClient.getQueryData<Paged<MatchItem>>(queryKeys.matches), conversationId)?.matchId;

  const confirmUnmatch = () => {
    const matchId = resolveMatchId();
    if (!matchId) {
      showToast("Impossible de retrouver ce match pour l'instant. Réessayez.", "error");
      return;
    }
    Alert.alert(
      `Supprimer le match avec ${firstName} ?`,
      "La conversation et tous ses messages seront supprimés pour vous deux.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Supprimer le match",
          style: "destructive",
          onPress: () => unmatch.mutate({ matchId, userId, conversationId })
        }
      ]
    );
  };

  const confirmHide = () => {
    Alert.alert("Masquer la conversation ?", "Elle réapparaîtra si un nouveau message arrive.", [
      { text: "Annuler", style: "cancel" },
      { text: "Masquer", onPress: () => hide.mutate() }
    ]);
  };

  const menu: ActionSheetOption[] = [
    { label: "Voir le profil", onPress: openProfile, testID: "chat-menu-profile" },
    { label: "Masquer la conversation", onPress: confirmHide, testID: "chat-menu-hide" },
    { label: "Supprimer le match", onPress: confirmUnmatch, destructive: true, testID: "chat-menu-unmatch" },
    { label: "Signaler", onPress: safety.openReport, destructive: true, testID: "chat-menu-report" },
    { label: "Bloquer", onPress: safety.confirmBlock, destructive: true, testID: "chat-menu-block" }
  ];

  const loadOlder = () => {
    if (thread.hasNextPage && !thread.isFetchingNextPage) {
      void thread.fetchNextPage();
    }
  };

  return (
    <SafeAreaView
      edges={["top", "bottom"]}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID="chat-screen"
    >
      <View style={[styles.header, { borderColor: colors.border }]}>
        <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} testID="chat-back" />
        <Pressable
          accessibilityHint="Ouvre le profil"
          accessibilityLabel={`Profil de ${firstName}`}
          accessibilityRole="button"
          onPress={openProfile}
          style={styles.person}
        >
          <Avatar size={40} thumbUrl={thumbUrl} />
          <Text numberOfLines={1} variant="label">
            {firstName}
          </Text>
        </Pressable>
        <IconButton
          icon="ellipsis-horizontal"
          label="Plus d'options"
          onPress={() => setMenuOpen(true)}
          testID="chat-menu-button"
        />
      </View>

      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {thread.isLoading ? (
          <View style={styles.center}>
            <ActivityIndicator color={colors.primary} />
          </View>
        ) : thread.isError ? (
          <ErrorState onRetry={() => void thread.refetch()} />
        ) : rows.length === 0 ? (
          <EmptyState
            icon="chatbubble-ellipses-outline"
            message={`Envoyez le premier message à ${firstName}.`}
            title="C'est un match !"
          />
        ) : (
          <FlatList
            ListFooterComponent={
              thread.isFetchingNextPage ? <ActivityIndicator color={colors.primary} style={styles.loader} /> : null
            }
            contentContainerStyle={styles.list}
            data={rows}
            inverted
            keyExtractor={keyExtractor}
            keyboardShouldPersistTaps="handled"
            onEndReached={loadOlder}
            onEndReachedThreshold={0.4}
            renderItem={renderRow}
            testID="chat-list"
          />
        )}

        <View style={[styles.composer, { borderColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            accessibilityLabel="Votre message"
            maxLength={MAX_MESSAGE_LENGTH}
            multiline
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textMuted}
            style={[styles.input, { backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }]}
            testID="chat-input"
            value={draft}
          />
          <IconButton
            color="primary"
            disabled={draft.trim().length === 0}
            icon="send"
            label="Envoyer"
            onPress={submit}
            testID="chat-send"
          />
        </View>
      </KeyboardAvoidingView>

      <ActionSheet onClose={() => setMenuOpen(false)} options={menu} title={firstName} visible={menuOpen} />
      {safety.reportModal}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    borderBottomWidth: StyleSheet.hairlineWidth,
    paddingHorizontal: spacing.xs
  },
  person: { flex: 1, minHeight: MIN_TOUCH_TARGET, flexDirection: "row", alignItems: "center", gap: spacing.md },
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  list: { paddingVertical: spacing.md },
  loader: { padding: spacing.lg },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: MIN_TOUCH_TARGET,
    borderRadius: radius.lg,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    paddingTop: 12,
    paddingBottom: 12,
    fontSize: 16
  }
});
