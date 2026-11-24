import React, { useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, Alert, FlatList, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import Ionicons from "@expo/vector-icons/Ionicons";
import { chatApi } from "../../api/endpoints";
import type { Message } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Avatar } from "../../components/Avatar";
import { Button } from "../../components/Button";
import { IconButton } from "../../components/IconButton";
import { Screen } from "../../components/Screen";
import { Sheet } from "../../components/Sheet";
import { EmptyState, ErrorState, LoadingState } from "../../components/States";
import { showToast } from "../../components/Toast";
import { useMessages, useSendMessage } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { queryKeys } from "../../queryClient";
import { colors, radii, spacing, type as typeScale } from "../../theme/tokens";
import { errorCode, errorMessage } from "../../utils/errors";
import { dayHeader, timeOfDay } from "../../utils/format";

type Row = { kind: "message"; message: Message; showDay: boolean; isLastMine: boolean };

export default function ChatScreen() {
  const navigation = useNavigation();
  const route = useRoute<RouteProp<RootStackParamList, "Chat">>();
  const { conversationId, matchId, userId, name, photoUrl } = route.params;
  const { account } = useAuth();
  const client = useQueryClient();
  const query = useMessages(conversationId);
  const send = useSendMessage(conversationId);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.messages) ?? [], [query.data]);

  // Mark as read whenever the newest message comes from the other member.
  const newest = messages[0];
  useEffect(() => {
    if (newest && newest.senderId !== account?.id && !newest.isRead) {
      chatApi
        .markRead(conversationId)
        .then(() => client.invalidateQueries({ queryKey: queryKeys.conversations }))
        .catch(() => undefined);
    }
  }, [newest, account?.id, conversationId, client]);

  const rows: Row[] = useMemo(() => {
    const lastMineIndex = messages.findIndex((m) => m.senderId === account?.id);
    return messages.map((message, index) => {
      const older = messages[index + 1];
      const showDay = !older || new Date(older.createdAt).toDateString() !== new Date(message.createdAt).toDateString();
      return { kind: "message", message, showDay, isLastMine: index === lastMineIndex };
    });
  }, [messages, account?.id]);

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    send.mutate(body, {
      onSuccess: () => setDraft(""),
      onError: (e) => showToast(errorMessage(e), "error")
    });
  };

  const hide = () => {
    const run = async () => {
      setMenu(false);
      try {
        await chatApi.hide(conversationId);
        await client.invalidateQueries({ queryKey: queryKeys.conversations });
        navigation.goBack();
      } catch (e) {
        showToast(errorMessage(e), "error");
      }
    };
    if (Platform.OS === "web") {
      if (globalThis.confirm?.("Supprimer cette conversation de ta liste ? Elle réapparaîtra si un nouveau message arrive.")) void run();
      return;
    }
    Alert.alert("Supprimer la conversation ?", "Elle disparaît de ta liste uniquement, et réapparaîtra si un nouveau message arrive.", [
      { text: "Annuler", style: "cancel" },
      { text: "Supprimer", style: "destructive", onPress: () => void run() }
    ]);
  };

  const openProfile = () => {
    setMenu(false);
    navigation.navigate("ProfileDetail", { userId, from: "chat", matchId, conversationId });
  };

  const header = (
    <View style={styles.header}>
      <IconButton icon="chevron-back" label="Retour" tone="plain" onPress={() => navigation.goBack()} testID="chat-back" />
      <Pressable style={styles.headerPeer} onPress={openProfile} accessibilityRole="button" accessibilityLabel={`Voir le profil de ${name}`}>
        <Avatar url={photoUrl} name={name} size={40} />
        <AppText variant="heading" numberOfLines={1} style={styles.flex}>
          {name}
        </AppText>
      </Pressable>
      <IconButton icon="ellipsis-horizontal" label="Options de la conversation" tone="plain" onPress={() => setMenu(true)} testID="chat-menu" />
    </View>
  );

  if (query.isError && errorCode(query.error) === "conversation_not_found") {
    return (
      <Screen>
        {header}
        <EmptyState icon="chatbubbles-outline" title="Conversation indisponible" message="Ce match a pris fin." actionLabel="Retour" onAction={() => navigation.goBack()} />
      </Screen>
    );
  }

  return (
    <Screen edges={["top", "bottom"]} testID="chat-screen">
      {header}
      {query.isPending ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          inverted
          data={rows}
          keyExtractor={(r) => r.message.id}
          contentContainerStyle={[styles.list, rows.length === 0 && styles.listEmpty]}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          ListFooterComponent={query.isFetchingNextPage ? <ActivityIndicator color={colors.primary} style={styles.loader} /> : null}
          ListEmptyComponent={
            <View style={styles.emptyChat}>
              <Avatar url={photoUrl} name={name} size={88} ring />
              <AppText variant="heading" align="center">
                Vous avez matché avec {name}
              </AppText>
              <AppText color={colors.textMuted} align="center">
                Lance la conversation avec une question sur son profil.
              </AppText>
            </View>
          }
          renderItem={({ item }) => {
            const m = item.message;
            const mine = m.senderId === account?.id;
            return (
              <View>
                {item.showDay ? (
                  <AppText variant="caption" color={colors.textSubtle} align="center" style={styles.day}>
                    {dayHeader(m.createdAt)}
                  </AppText>
                ) : null}
                <View style={[styles.bubbleRow, mine ? styles.rowMine : styles.rowTheirs]}>
                  <View style={[styles.bubble, mine ? styles.mine : styles.theirs]} testID="chat-message">
                    <AppText color={mine ? colors.onPrimary : colors.text}>{m.body}</AppText>
                  </View>
                  <AppText variant="caption" color={colors.textSubtle} style={styles.meta}>
                    {timeOfDay(m.createdAt)}
                    {mine && item.isLastMine ? (m.isRead ? " · Lu" : " · Envoyé") : ""}
                  </AppText>
                </View>
              </View>
            );
          }}
        />
      )}
      <View style={styles.composer}>
        <TextInput
          ref={inputRef}
          value={draft}
          onChangeText={setDraft}
          placeholder="Écris ton message…"
          placeholderTextColor={colors.textSubtle}
          multiline
          maxLength={2000}
          style={styles.input}
          accessibilityLabel="Message"
          testID="chat-input"
          onKeyPress={(e) => {
            // Enter sends on web; Shift+Enter adds a new line.
            const native = e.nativeEvent as unknown as { key: string; shiftKey?: boolean };
            if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
        />
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Envoyer"
          onPress={submit}
          disabled={!draft.trim() || send.isPending}
          style={[styles.send, (!draft.trim() || send.isPending) && styles.sendDisabled]}
          testID="chat-send"
        >
          {send.isPending ? <ActivityIndicator color={colors.onPrimary} /> : <Ionicons name="arrow-up" size={22} color={colors.onPrimary} />}
        </Pressable>
      </View>

      <Sheet visible={menu} title={name} onClose={() => setMenu(false)}>
        <Button label="Voir le profil" icon="person-outline" variant="secondary" onPress={openProfile} />
        <Button label="Supprimer la conversation" icon="trash-outline" variant="secondary" onPress={hide} testID="chat-hide" />
        <AppText variant="caption" color={colors.textMuted}>
          Pour annuler le match, bloquer ou signaler {name}, ouvre son profil.
        </AppText>
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", gap: spacing.xs, paddingHorizontal: spacing.sm, paddingVertical: spacing.xs, borderBottomWidth: 1, borderBottomColor: colors.border },
  headerPeer: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  flex: { flex: 1 },
  list: { paddingHorizontal: spacing.md, paddingVertical: spacing.md, gap: spacing.xs },
  listEmpty: { flexGrow: 1, justifyContent: "center" },
  emptyChat: { alignItems: "center", gap: spacing.sm, padding: spacing.xl, transform: [{ scaleY: -1 }] },
  loader: { marginVertical: spacing.md },
  day: { marginVertical: spacing.sm },
  bubbleRow: { maxWidth: "82%", gap: 2, marginVertical: 2 },
  rowMine: { alignSelf: "flex-end", alignItems: "flex-end" },
  rowTheirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.lg },
  mine: { backgroundColor: colors.primary, borderBottomRightRadius: 6 },
  theirs: { backgroundColor: colors.surfaceRaised, borderBottomLeftRadius: 6 },
  meta: { marginHorizontal: spacing.xs },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderTopWidth: 1, borderTopColor: colors.border },
  input: {
    flex: 1,
    maxHeight: 140,
    minHeight: 44,
    color: colors.text,
    backgroundColor: colors.surface,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    paddingHorizontal: spacing.md,
    paddingTop: spacing.sm,
    paddingBottom: spacing.sm,
    ...typeScale.body,
    ...Platform.select({ web: { outlineWidth: 0 }, default: {} })
  },
  send: { width: 44, height: 44, borderRadius: 22, backgroundColor: colors.primary, alignItems: "center", justifyContent: "center" },
  sendDisabled: { opacity: 0.4 }
});
