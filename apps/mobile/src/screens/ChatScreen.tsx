import React, { useCallback, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useIsFocused } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { hideConversation } from "../api/chat";
import { fetchMatches, unmatch } from "../api/discovery";
import { blockUser, reportUser } from "../api/safety";
import type { ReportReason } from "../api/types";
import { ActionSheet, ConfirmDialog, ReportDialog } from "../components/Dialogs";
import { PersonAvatar } from "../components/PersonAvatar";
import { useAuth } from "../hooks/useAuth";
import { useChat, type ChatMessage } from "../hooks/useChat";
import { useRealtimeEvents } from "../hooks/useRealtime";
import { formatClock, formatDayHeading, isSameDay } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, MessageItem, Notice, Text, colors, radii, spacing } from "../shared/ui";

type Dialog = "menu" | "report" | "block" | "unmatch" | "hide" | null;

export default function ChatScreen() {
  const route = useRoute<RouteProp<RootStackParamList, "Chat">>();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const focused = useIsFocused();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { conversationId, peerId, name } = route.params;
  const chat = useChat(conversationId, user?.id ?? "", focused);

  const [draft, setDraft] = useState("");
  const [dialog, setDialog] = useState<Dialog>(null);
  const [actionBusy, setActionBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  // Needed to unmatch: find the match that owns this conversation.
  const matches = useQuery({ queryKey: ["matches"], queryFn: () => fetchMatches(), staleTime: 60_000 });
  const photo = matches.data?.matches.find((m) => m.conversationId === conversationId)?.user.photo;

  // The other person unmatched/blocked us: leave the conversation.
  useRealtimeEvents((event) => {
    if (event.type === "match.removed" && event.data.conversationId === conversationId) {
      navigation.navigate("Tabs", { screen: "Messages" });
    }
  });

  const closeAndRefresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["conversations"] });
    await queryClient.invalidateQueries({ queryKey: ["matches"] });
    await queryClient.invalidateQueries({ queryKey: ["discover"] });
    navigation.navigate("Tabs", { screen: "Messages" });
  }, [navigation, queryClient]);

  const perform = async (task: () => Promise<void>) => {
    setActionBusy(true);
    setActionError(null);
    try {
      await task();
      setDialog(null);
      await closeAndRefresh();
    } catch (err) {
      setActionError(errorMessage(err));
    } finally {
      setActionBusy(false);
    }
  };

  const submit = async () => {
    const text = draft;
    if (!text.trim()) {
      return;
    }
    setDraft("");
    try {
      await chat.send(text);
    } catch {
      setDraft(text);
    }
  };

  // Newest first, because the list is inverted.
  const rows = useMemo(() => [...chat.messages].reverse(), [chat.messages]);
  const lastMineId = useMemo(
    () => [...chat.messages].reverse().find((m) => m.senderId === user?.id && !m.pending)?.id,
    [chat.messages, user?.id]
  );

  const renderItem = ({ item, index }: { item: ChatMessage; index: number }) => {
    const mine = item.senderId === user?.id;
    const older = rows[index + 1];
    const showDay = !older || !isSameDay(older.createdAt, item.createdAt);
    const meta = item.pending
      ? "Envoi…"
      : mine && item.id === lastMineId && item.readAt
        ? `${formatClock(item.createdAt)} · Lu`
        : formatClock(item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text style={styles.day} tone="muted" variant="caption" weight="semibold">
            {formatDayHeading(item.createdAt)}
          </Text>
        ) : null}
        <MessageItem content={item.content} meta={meta} mine={mine} />
      </View>
    );
  };

  return (
    <SafeAreaLayout edges={["top", "left", "right", "bottom"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <View style={styles.header}>
          <Pressable
            accessibilityLabel="Retour"
            accessibilityRole="button"
            onPress={() => navigation.goBack()}
            style={styles.headerButton}
            testID="chat-back"
          >
            <Text variant="heading">←</Text>
          </Pressable>
          <Pressable
            accessibilityRole="button"
            onPress={() => navigation.navigate("ProfileDetail", { userId: peerId, conversationId })}
            style={styles.headerPeer}
          >
            <PersonAvatar name={name} photo={photo} size={40} />
            <Text variant="heading" weight="bold" numberOfLines={1}>
              {name}
            </Text>
          </Pressable>
          <Pressable
            accessibilityLabel="Plus d'options"
            accessibilityRole="button"
            onPress={() => setDialog("menu")}
            style={styles.headerButton}
            testID="chat-menu"
          >
            <Text variant="heading">⋯</Text>
          </Pressable>
        </View>

        {chat.loading && chat.messages.length === 0 ? (
          <LoadingView label="Chargement des messages…" />
        ) : chat.error && chat.messages.length === 0 ? (
          <ErrorView actionLabel="Réessayer" message={chat.error} onAction={chat.reload} />
        ) : (
          <FlatList
            ListEmptyComponent={
              <View style={[styles.empty, styles.invertFix]}>
                <Text style={styles.centerText} tone="muted">
                  Vous avez matché avec {name}. Dites bonjour !
                </Text>
              </View>
            }
            contentContainerStyle={styles.list}
            data={rows}
            inverted
            keyExtractor={(item) => item.id}
            onEndReached={chat.hasMore ? chat.loadMore : undefined}
            onEndReachedThreshold={0.4}
            renderItem={renderItem}
            testID="chat-messages"
          />
        )}

        {chat.sendError ? <Notice description={chat.sendError} style={styles.notice} title="Message non envoyé" tone="danger" /> : null}
        <View style={styles.composer}>
          <TextInput
            accessibilityLabel="Votre message"
            maxLength={2000}
            multiline
            onChangeText={setDraft}
            placeholder="Écrivez un message…"
            placeholderTextColor={colors.textMuted}
            style={styles.input}
            testID="chat-input"
            value={draft}
          />
          <Button disabled={!draft.trim()} label="Envoyer" onPress={submit} size="sm" testID="chat-send" />
        </View>
      </KeyboardAvoidingView>

      <ActionSheet
        onClose={() => setDialog(null)}
        options={[
          { label: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId: peerId, conversationId }) },
          { label: "Masquer cette conversation", onPress: () => setDialog("hide") },
          { label: "Annuler le match", onPress: () => setDialog("unmatch"), destructive: true },
          { label: "Signaler", onPress: () => setDialog("report"), destructive: true, testID: "chat-report" },
          { label: "Bloquer", onPress: () => setDialog("block"), destructive: true, testID: "chat-block" }
        ]}
        title={name}
        visible={dialog === "menu"}
      />
      <ConfirmDialog
        confirmLabel="Masquer"
        error={actionError}
        loading={actionBusy}
        message="Les messages actuels disparaîtront de votre côté uniquement. Ils réapparaîtront si un nouveau message arrive, sans l'historique."
        onCancel={() => setDialog(null)}
        onConfirm={() => perform(() => hideConversation(conversationId))}
        title="Masquer la conversation ?"
        visible={dialog === "hide"}
      />
      <ConfirmDialog
        confirmLabel="Annuler le match"
        destructive
        error={actionError}
        loading={actionBusy}
        message={`Votre match avec ${name} et la conversation seront supprimés pour vous deux.`}
        onCancel={() => setDialog(null)}
        onConfirm={() =>
          perform(async () => {
            const match = matches.data?.matches.find((m) => m.conversationId === conversationId);
            if (!match) {
              throw new Error("Match introuvable.");
            }
            await unmatch(match.id);
          })
        }
        title="Annuler ce match ?"
        visible={dialog === "unmatch"}
      />
      <ConfirmDialog
        confirmLabel="Bloquer"
        destructive
        error={actionError}
        loading={actionBusy}
        message={`${name} ne pourra plus vous voir ni vous écrire, et la conversation sera supprimée.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => perform(() => blockUser(peerId))}
        title={`Bloquer ${name} ?`}
        visible={dialog === "block"}
      />
      <ReportDialog
        error={actionError}
        loading={actionBusy}
        name={name}
        onCancel={() => setDialog(null)}
        onSubmit={(reason: ReportReason, details, alsoBlock) => perform(() => reportUser(peerId, reason, details, alsoBlock))}
        visible={dialog === "report"}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    gap: spacing.xs
  },
  headerButton: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  headerPeer: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { padding: spacing.lg, flexGrow: 1 },
  day: { textAlign: "center", marginVertical: spacing.md },
  empty: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  invertFix: { transform: [{ scaleY: -1 }] },
  centerText: { textAlign: "center" },
  notice: { marginHorizontal: spacing.lg, marginBottom: spacing.sm },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    backgroundColor: colors.backgroundElevated,
    borderTopWidth: 1,
    borderTopColor: colors.border
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.background,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    color: colors.text,
    fontSize: 15
  }
});
