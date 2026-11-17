import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { FlatList, Modal, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import { Ionicons } from "@expo/vector-icons";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { errorMessage, isApiError } from "../../api/client";
import { chatApi, discoveryApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Message, MessagesPage } from "../../api/types";
import { useCurrentUser } from "../../auth/AuthProvider";
import { ActiveConversationContext } from "../../navigation/RootNavigator";
import type { RootScreenProps } from "../../navigation/types";
import { insertMessage } from "../../realtime/cache";
import { colors, radii, spacing, typography } from "../../theme";
import { Avatar, Chip, EmptyState, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { clockTime, dayLabel } from "../../utils/format";
import { confirmAction } from "../../utils/confirm";
import { useBlockUser } from "../safety/useSafety";

const ICEBREAKERS = [
  "Salut ! Comment se passe ta semaine ?",
  "Ton dimanche idéal, ça ressemble à quoi ?",
  "Café ou balade pour un premier rendez-vous ?"
];
const MAX_LENGTH = 2000;

type Row = { kind: "message"; message: Message } | { kind: "day"; label: string; key: string };

export function ConversationScreen({ navigation, route }: RootScreenProps<"Conversation">) {
  const { conversationId } = route.params;
  const me = useCurrentUser();
  const queryClient = useQueryClient();
  const activeRef = useContext(ActiveConversationContext);
  const [draft, setDraft] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const conversation = useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => chatApi.conversation(conversationId) });
  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    enabled: conversation.isSuccess
  });

  const other = conversation.data?.profile;
  const block = useBlockUser(() => navigation.popToTop());

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () =>
        other ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Voir le profil de ${other.firstName}`}
            onPress={() => navigation.navigate("ProfileDetail", { userId: other.userId })}
            style={styles.headerTitle}
          >
            <Avatar url={other.photos[0]?.url} name={other.firstName} size={34} />
            <Text variant="bodyStrong">{other.firstName}</Text>
          </Pressable>
        ) : null,
      headerRight: () =>
        other ? (
          <IconButton
            icon="ellipsis-horizontal"
            label="Options de la conversation"
            testID="conversation-menu"
            size={38}
            onPress={() => setMenuOpen(true)}
          />
        ) : null
    });
  }, [navigation, other]);

  // Mark as read while the screen is focused, and whenever a new incoming message arrives.
  const newestIncoming = messages.data?.pages[0]?.messages.find((m) => m.senderId !== me.id)?.id;
  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadConversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  });
  const { mutate: markReadMutate } = markRead;
  useFocusEffect(
    useCallback(() => {
      activeRef.current = conversationId;
      return () => {
        activeRef.current = null;
      };
    }, [activeRef, conversationId])
  );
  useEffect(() => {
    if (conversation.isSuccess) markReadMutate();
  }, [conversation.isSuccess, newestIncoming, markReadMutate]);

  const send = useMutation({
    mutationFn: (message: Message) => chatApi.send(conversationId, message.content, message.clientMessageId ?? ""),
    onMutate: (message) => {
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => insertMessage(data, message));
    },
    onSuccess: (saved) => {
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => insertMessage(data, saved));
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (error, message) => {
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) =>
        data ? { ...data, pages: data.pages.map((p) => ({ ...p, messages: p.messages.filter((m) => m.id !== message.id) })) } : data
      );
      setDraft(message.content);
      showToast(errorMessage(error), "error");
    }
  });

  function submit(text = draft) {
    const content = text.trim();
    if (!content || content.length > MAX_LENGTH) return;
    const clientMessageId = `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
    send.mutate({
      id: `local-${clientMessageId}`,
      clientMessageId,
      conversationId,
      senderId: me.id,
      content,
      createdAt: new Date().toISOString()
    });
    setDraft("");
  }

  const rows = useMemo<Row[]>(() => {
    const list = messages.data?.pages.flatMap((p) => p.messages) ?? [];
    const out: Row[] = [];
    list.forEach((m, i) => {
      out.push({ kind: "message", message: m });
      const older = list[i + 1];
      const label = dayLabel(m.createdAt);
      if (!older || dayLabel(older.createdAt) !== label) out.push({ kind: "day", label, key: `day-${m.id}` });
    });
    return out;
  }, [messages.data]);

  const otherReadAt = conversation.data?.otherLastReadAt ? new Date(conversation.data.otherLastReadAt).getTime() : 0;
  const lastMineId = messages.data?.pages[0]?.messages.find((m) => m.senderId === me.id)?.id;

  async function runMenu(action: "profile" | "hide" | "unmatch" | "block" | "report") {
    setMenuOpen(false);
    if (!other || !conversation.data) return;
    try {
      if (action === "profile") navigation.navigate("ProfileDetail", { userId: other.userId });
      if (action === "report") navigation.navigate("Report", { userId: other.userId, firstName: other.firstName });
      if (action === "block") await block.confirm(other.userId, other.firstName);
      if (
        action === "hide" &&
        (await confirmAction("Supprimer la conversation ?", "Les messages seront effacés de ton côté uniquement.", "Supprimer"))
      ) {
        await chatApi.hide(conversationId);
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        navigation.goBack();
      }
      if (
        action === "unmatch" &&
        (await confirmAction(
          `Annuler le match avec ${other.firstName} ?`,
          "La conversation sera fermée pour vous deux.",
          "Annuler le match"
        ))
      ) {
        await discoveryApi.unmatch(conversation.data.matchId);
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        navigation.goBack();
      }
    } catch (error) {
      showToast(errorMessage(error), "error");
    }
  }

  if (conversation.isLoading)
    return (
      <Screen edges={["left", "right"]}>
        <LoadingState />
      </Screen>
    );
  if (conversation.isError) {
    const gone = isApiError(conversation.error, "conversation_not_found");
    return (
      <Screen edges={["left", "right"]}>
        <EmptyState
          icon={gone ? "chatbubble-ellipses-outline" : "cloud-offline-outline"}
          title={gone ? "Conversation indisponible" : "Chargement impossible"}
          message={gone ? "Ce match a pris fin." : errorMessage(conversation.error)}
          actionLabel={gone ? "Retour" : "Réessayer"}
          onAction={() => (gone ? navigation.goBack() : void conversation.refetch())}
        />
      </Screen>
    );
  }

  const empty = messages.isSuccess && rows.length === 0;
  return (
    <Screen edges={["left", "right", "bottom"]} contentStyle={styles.noPad} testID="conversation-screen">
      <FlatList
        inverted={!empty}
        data={rows}
        keyExtractor={(r) => (r.kind === "day" ? r.key : r.message.id)}
        onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
        onEndReachedThreshold={0.3}
        contentContainerStyle={[styles.list, empty ? styles.emptyList : null]}
        ListEmptyComponent={
          messages.isLoading ? (
            <LoadingState />
          ) : (
            <View style={styles.icebreak}>
              <Avatar url={other?.photos[0]?.url} name={other?.firstName ?? ""} size={88} ring />
              <Text variant="heading" center>
                Vous avez matché !
              </Text>
              <Text tone="muted" center>
                Lance la conversation avec {other?.firstName}. Une idée :
              </Text>
              <View style={styles.icebreakers}>
                {ICEBREAKERS.map((text) => (
                  <Chip key={text} label={text} onPress={() => setDraft(text)} />
                ))}
              </View>
            </View>
          )
        }
        renderItem={({ item }) => {
          if (item.kind === "day") {
            return (
              <Text variant="caption" tone="subtle" center style={styles.day}>
                {item.label}
              </Text>
            );
          }
          const m = item.message;
          const mine = m.senderId === me.id;
          const pending = m.id.startsWith("local-");
          const read = mine && otherReadAt >= new Date(m.createdAt).getTime();
          return (
            <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
              <View style={[styles.bubble, mine ? styles.mine : styles.theirs]} testID="message-bubble">
                <Text tone={mine ? "inverse" : "default"} selectable>
                  {m.content}
                </Text>
                <Text variant="caption" tone={mine ? "inverse" : "subtle"} style={styles.time}>
                  {clockTime(m.createdAt)}
                </Text>
              </View>
              {m.id === lastMineId || pending ? (
                <Text variant="caption" tone="subtle" style={styles.status} testID="message-status">
                  {pending ? "Envoi…" : read ? "Lu" : "Envoyé"}
                </Text>
              ) : null}
            </View>
          );
        }}
      />
      <View style={styles.composer}>
        <TextInput
          ref={inputRef}
          testID="message-input"
          accessibilityLabel="Écrire un message"
          placeholder="Écris un message…"
          placeholderTextColor={colors.textSubtle}
          selectionColor={colors.accent}
          value={draft}
          onChangeText={setDraft}
          multiline
          maxLength={MAX_LENGTH}
          style={styles.input}
          onKeyPress={(e) => {
            const native = e.nativeEvent as { key: string; shiftKey?: boolean };
            if (native.key === "Enter" && !native.shiftKey) {
              (e as unknown as { preventDefault?: () => void }).preventDefault?.();
              submit();
            }
          }}
        />
        <IconButton
          icon="arrow-up"
          label="Envoyer"
          testID="message-send"
          size={44}
          color={colors.accentText}
          background={colors.accent}
          borderColor={colors.accent}
          disabled={!draft.trim()}
          onPress={() => submit()}
        />
      </View>

      <Modal visible={menuOpen} transparent animationType="fade" onRequestClose={() => setMenuOpen(false)}>
        <Pressable style={styles.sheetBackdrop} onPress={() => setMenuOpen(false)} accessibilityLabel="Fermer le menu">
          <View style={styles.sheet}>
            <MenuItem icon="person-outline" label="Voir le profil" onPress={() => void runMenu("profile")} />
            <MenuItem
              icon="trash-outline"
              label="Supprimer la conversation (pour moi)"
              onPress={() => void runMenu("hide")}
              testID="menu-hide"
            />
            <MenuItem icon="heart-dislike-outline" label="Annuler le match" onPress={() => void runMenu("unmatch")} testID="menu-unmatch" />
            <MenuItem icon="flag-outline" label="Signaler" onPress={() => void runMenu("report")} testID="menu-report" />
            <MenuItem icon="ban-outline" label="Bloquer" destructive onPress={() => void runMenu("block")} testID="menu-block" />
          </View>
        </Pressable>
      </Modal>
    </Screen>
  );
}

function MenuItem({
  icon,
  label,
  onPress,
  destructive,
  testID
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  destructive?: boolean;
  testID?: string;
}) {
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.menuItem, pressed ? styles.menuPressed : null]}
    >
      <Ionicons name={icon} size={20} color={destructive ? colors.danger : colors.text} />
      <Text tone={destructive ? "danger" : "default"}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  noPad: {
    paddingHorizontal: 0,
    paddingTop: 0
  },
  headerTitle: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  list: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  emptyList: {
    flexGrow: 1,
    justifyContent: "center"
  },
  icebreak: {
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg
  },
  icebreakers: {
    gap: spacing.sm,
    alignItems: "center"
  },
  day: {
    marginVertical: spacing.md
  },
  bubbleRow: {
    alignItems: "flex-start",
    marginVertical: 3
  },
  mineRow: {
    alignItems: "flex-end"
  },
  bubble: {
    maxWidth: "80%",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg,
    gap: 2
  },
  mine: {
    backgroundColor: colors.accent,
    borderBottomRightRadius: radii.sm
  },
  theirs: {
    backgroundColor: colors.surfaceRaised,
    borderBottomLeftRadius: radii.sm
  },
  time: {
    alignSelf: "flex-end",
    opacity: 0.75
  },
  status: {
    marginTop: 2,
    marginRight: spacing.xs
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border
  },
  input: {
    ...typography.body,
    flex: 1,
    color: colors.text,
    maxHeight: 140,
    minHeight: 44,
    paddingHorizontal: spacing.lg,
    paddingTop: 10,
    paddingBottom: 10,
    borderRadius: radii.xl,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  sheetBackdrop: {
    flex: 1,
    backgroundColor: colors.scrim,
    justifyContent: "flex-end"
  },
  sheet: {
    backgroundColor: colors.surfaceRaised,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    paddingVertical: spacing.md,
    paddingBottom: spacing.xxl
  },
  menuItem: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.lg
  },
  menuPressed: {
    backgroundColor: colors.surfacePressed
  }
});
