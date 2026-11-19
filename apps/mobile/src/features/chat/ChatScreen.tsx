import React, { useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { chatApi, discoveryApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Message } from "../../api/types";
import { formatDayLabel, formatTime } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { addMessageToCache } from "../../state/realtime";
import { useTheme } from "../../theme/ThemeProvider";
import { fonts, radii, spacing } from "../../theme/tokens";
import { ActionSheet, Avatar, EmptyState, Icon, IconButton, LoadingState, Text, useFeedback } from "../../ui";
import { ReportSheet } from "../safety/ReportSheet";
import { useBlock } from "../safety/useBlock";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

type Row = { kind: "message"; message: Message; showReceipt: boolean } | { kind: "day"; id: string; label: string };

const MAX_LENGTH = 2000;

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId } = route.params;
  const { colors } = useTheme();
  const { toast, confirm } = useFeedback();
  const client = useQueryClient();
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const conversation = useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => chatApi.get(conversationId) });
  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    enabled: conversation.isSuccess,
    staleTime: Infinity
  });
  const partner = conversation.data?.partner;
  const goBack = () => (navigation.canGoBack() ? navigation.goBack() : navigation.navigate("Tabs", { screen: "Chats" }));
  const block = useBlock(goBack);

  const list = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);

  // Newest first (inverted list) with day separators placed after the oldest message of each day.
  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [];
    const lastMineIndex = list.findIndex((m) => m.isMine);
    list.forEach((m, i) => {
      out.push({ kind: "message", message: m, showReceipt: i === lastMineIndex && i === 0 });
      const older = list[i + 1];
      const day = new Date(m.createdAt).toDateString();
      if (!older || new Date(older.createdAt).toDateString() !== day) {
        out.push({ kind: "day", id: `day-${m.id}`, label: formatDayLabel(m.createdAt) });
      }
    });
    return out;
  }, [list]);

  // Mark as read whenever an incoming message is displayed.
  const newestIncoming = list.find((m) => !m.isMine)?.id;
  useEffect(() => {
    if (!conversation.isSuccess) return;
    void chatApi
      .markRead(conversationId)
      .then(() => {
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
        void client.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
      })
      .catch(() => undefined);
  }, [conversationId, newestIncoming, conversation.isSuccess, client]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      addMessageToCache(client, message);
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (error, body) => {
      setDraft((current) => current || body);
      toast(errorMessage(error), "error");
    }
  });

  const unmatch = useMutation({
    mutationFn: () => discoveryApi.unmatch(conversation.data?.matchId ?? ""),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      toast("Match annulé", "success");
      goBack();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      client.removeQueries({ queryKey: queryKeys.messages(conversationId) });
      toast("Conversation supprimée de votre appareil", "success");
      goBack();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body);
  };

  if (conversation.isLoading) {
    return (
      <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]}>
        <LoadingState />
      </SafeAreaView>
    );
  }
  if (conversation.error || !partner) {
    return (
      <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]}>
        <View style={styles.headerRow}>
          <IconButton icon="chevron-back" label="Retour" onPress={goBack} />
        </View>
        <EmptyState
          icon="lock-closed-outline"
          title="Conversation indisponible"
          message="Ce match a pris fin ou la conversation n'existe plus."
          actionLabel="Retour aux messages"
          onAction={goBack}
        />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]} edges={["top", "left", "right", "bottom"]}>
      <View style={[styles.headerRow, { borderBottomColor: colors.border }]}>
        <IconButton icon="chevron-back" label="Retour" onPress={goBack} />
        <Pressable
          style={styles.headerPerson}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${partner.firstName}`}
          onPress={() => navigation.navigate("ProfileDetail", { userId: partner.userId })}
        >
          <Avatar path={partner.photoUrl} name={partner.firstName} size={38} />
          <Text variant="heading" numberOfLines={1}>
            {partner.firstName}
          </Text>
        </Pressable>
        <IconButton icon="ellipsis-horizontal" label="Options de la conversation" onPress={() => setMenu(true)} testID="chat-menu" />
      </View>
      <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {messages.isLoading ? (
          <LoadingState />
        ) : (
          <FlatList
            inverted
            data={rows}
            keyExtractor={(r) => (r.kind === "day" ? r.id : r.message.id)}
            contentContainerStyle={styles.messages}
            onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
            onEndReachedThreshold={0.3}
            ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={colors.primary} /> : null}
            ListEmptyComponent={
              <View style={styles.emptyWrap}>
                <Avatar path={partner.photoUrl} name={partner.firstName} size={88} ring />
                <Text variant="heading" align="center">
                  Vous avez matché avec {partner.firstName}
                </Text>
                <Text tone="muted" align="center">
                  Une question sur ses centres d’intérêt est souvent un bon début.
                </Text>
              </View>
            }
            renderItem={({ item }) =>
              item.kind === "day" ? (
                <Text variant="caption" tone="subtle" align="center" style={styles.day}>
                  {item.label}
                </Text>
              ) : (
                <Bubble message={item.message} showReceipt={item.showReceipt} />
              )
            }
          />
        )}
        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            ref={inputRef}
            value={draft}
            onChangeText={setDraft}
            placeholder={`Écrire à ${partner.firstName}…`}
            placeholderTextColor={colors.textSubtle}
            multiline
            maxLength={MAX_LENGTH}
            accessibilityLabel="Votre message"
            style={[
              styles.input,
              { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border, fontFamily: fonts.body }
            ]}
            onKeyPress={(e) => {
              // Web: Enter sends, Shift+Enter adds a line.
              const native = e.nativeEvent as unknown as { key: string; shiftKey?: boolean };
              if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                (e as unknown as { preventDefault: () => void }).preventDefault();
                submit();
              }
            }}
            testID="chat-input"
          />
          <IconButton
            icon="arrow-up"
            label="Envoyer"
            tone="primary"
            size={44}
            disabled={!draft.trim() || send.isPending}
            onPress={submit}
            testID="chat-send"
          />
        </View>
      </KeyboardAvoidingView>
      <ActionSheet
        visible={menu}
        title={partner.firstName}
        onClose={() => setMenu(false)}
        actions={[
          {
            label: "Voir le profil",
            icon: "person-outline",
            onPress: () => navigation.navigate("ProfileDetail", { userId: partner.userId })
          },
          {
            label: "Supprimer la conversation (pour moi)",
            icon: "trash-outline",
            onPress: async () => {
              if (
                await confirm({
                  title: "Supprimer l'historique ?",
                  message: "Les messages disparaîtront de votre côté uniquement.",
                  confirmLabel: "Supprimer",
                  destructive: true
                })
              )
                clear.mutate();
            }
          },
          {
            label: "Annuler le match",
            icon: "heart-dislike-outline",
            destructive: true,
            onPress: async () => {
              if (
                await confirm({
                  title: `Annuler le match avec ${partner.firstName} ?`,
                  message: "Vous ne pourrez plus vous écrire.",
                  confirmLabel: "Annuler le match",
                  destructive: true
                })
              )
                unmatch.mutate();
            }
          },
          { label: "Signaler", icon: "flag-outline", destructive: true, onPress: () => setReporting(true) },
          { label: "Bloquer", icon: "ban-outline", destructive: true, onPress: () => void block(partner.userId, partner.firstName) }
        ]}
      />
      <ReportSheet
        userId={partner.userId}
        name={partner.firstName}
        visible={reporting}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          goBack();
        }}
      />
    </SafeAreaView>
  );
}

function Bubble({ message, showReceipt }: { message: Message; showReceipt: boolean }) {
  const { colors } = useTheme();
  const mine = message.isMine;
  return (
    <View style={[styles.bubbleWrap, mine ? styles.mineWrap : styles.theirsWrap]} testID={mine ? "message-mine" : "message-theirs"}>
      <View
        style={[
          styles.bubble,
          mine
            ? { backgroundColor: colors.primary, borderBottomRightRadius: 6 }
            : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1, borderBottomLeftRadius: 6 }
        ]}
      >
        <Text style={{ color: mine ? colors.onPrimary : colors.text }} selectable>
          {message.body}
        </Text>
        <Text variant="caption" style={[styles.time, { color: mine ? colors.onPrimary : colors.textSubtle }]}>
          {formatTime(message.createdAt)}
        </Text>
      </View>
      {showReceipt ? (
        <View style={styles.receipt}>
          <Icon
            name={message.isRead ? "checkmark-done" : "checkmark"}
            size={14}
            color={message.isRead ? colors.primary : colors.textSubtle}
          />
          <Text variant="caption" tone="subtle" testID="read-receipt">
            {message.isRead ? "Lu" : "Envoyé"}
          </Text>
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  headerRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  headerPerson: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  messages: { padding: spacing.md, gap: 6, flexGrow: 1, maxWidth: 760, width: "100%", alignSelf: "center" },
  emptyWrap: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl, transform: [{ scaleY: -1 }] },
  day: { marginVertical: spacing.sm },
  bubbleWrap: { maxWidth: "80%" },
  mineWrap: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirsWrap: { alignSelf: "flex-start" },
  bubble: { borderRadius: 20, paddingHorizontal: 14, paddingVertical: 9, gap: 2 },
  time: { alignSelf: "flex-end", opacity: 0.75, fontSize: 11 },
  receipt: { flexDirection: "row", alignItems: "center", gap: 3, marginTop: 3 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 130,
    borderWidth: 1,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.md,
    paddingTop: 11,
    paddingBottom: 11,
    fontSize: 16
  }
});
