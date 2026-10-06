import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { MatchSummary } from "../api/types";
import { useAuth } from "../hooks/useAuth";
import { useMatches } from "../hooks/queries";
import { describeError } from "../lib/errors";
import { formatRelative, previewText } from "../lib/format";
import type { RootStackParamList, TabParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Avatar, Text, spacing, useTheme } from "../shared/ui";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Messages">, NativeStackScreenProps<RootStackParamList>>;

export default function MatchesScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { user } = useAuth();
  const matches = useMatches();

  const open = (match: MatchSummary) =>
    navigation.navigate("Chat", { matchId: match.id, peerId: match.user.userId, peerName: match.user.firstName });

  const all = matches.data?.matches ?? [];
  const fresh = all.filter((match) => !match.lastMessage);
  const conversations = all.filter((match) => match.lastMessage);

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <View style={styles.padded}>
        <Header title="Messages" />
      </View>

      {matches.isLoading ? (
        <LoadingView fullScreen label="Chargement…" />
      ) : matches.isError ? (
        <ErrorView message={describeError(matches.error)} onAction={() => void matches.refetch()} />
      ) : all.length === 0 ? (
        <EmptyView
          icon="chatbubbles-outline"
          message="Likez des profils : dès qu'un like est réciproque, la conversation s'ouvre ici."
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.freshBlock}>
                <Text tone="secondary" variant="eyebrow" weight="bold">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.fresh} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((match) => (
                    <Pressable
                      accessibilityLabel={`Nouveau match avec ${match.user.firstName}`}
                      accessibilityRole="button"
                      key={match.id}
                      onPress={() => open(match)}
                      style={styles.freshItem}
                      testID={`new-match-${match.id}`}
                    >
                      <Avatar name={match.user.firstName} photoPath={match.user.primaryPhotoUrl} size={68} />
                      <Text numberOfLines={1} variant="label" weight="semibold">
                        {match.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {conversations.length > 0 ? (
                  <Text style={styles.sectionGap} tone="secondary" variant="eyebrow" weight="bold">
                    Conversations
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          contentContainerStyle={styles.list}
          data={conversations}
          keyExtractor={(match) => match.id}
          refreshControl={<RefreshControl onRefresh={() => void matches.refetch()} refreshing={matches.isRefetching} />}
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount > 0 ? `, ${item.unreadCount} non lus` : ""}`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, { borderBottomColor: colors.border, opacity: pressed ? 0.7 : 1 }]}
                testID={`conversation-${item.id}`}
              >
                <Avatar name={item.user.firstName} photoPath={item.user.primaryPhotoUrl} size={56} />
                <View style={styles.rowCopy}>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} weight={item.unreadCount > 0 ? "bold" : "semibold"}>
                      {item.user.firstName}
                    </Text>
                    <Text tone="subtle" variant="caption">
                      {last ? formatRelative(last.createdAt) : ""}
                    </Text>
                  </View>
                  <View style={styles.rowTop}>
                    <Text
                      numberOfLines={1}
                      style={styles.flex}
                      tone={item.unreadCount > 0 ? "default" : "muted"}
                      weight={item.unreadCount > 0 ? "semibold" : "regular"}
                    >
                      {last ? `${mine ? "Vous : " : ""}${previewText(last.body)}` : ""}
                    </Text>
                    {item.unreadCount > 0 ? (
                      <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                        <Text style={{ color: colors.primaryForeground, fontSize: 11, lineHeight: 14 }} weight="bold">
                          {item.unreadCount > 99 ? "99+" : item.unreadCount}
                        </Text>
                      </View>
                    ) : null}
                  </View>
                </View>
              </Pressable>
            );
          }}
          testID="matches-list"
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.xl, paddingTop: spacing.md },
  list: { paddingHorizontal: spacing.xl, paddingBottom: spacing.xl },
  freshBlock: { gap: spacing.sm, marginBottom: spacing.sm },
  fresh: { gap: spacing.lg, paddingVertical: spacing.sm },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  sectionGap: { marginTop: spacing.md },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  rowCopy: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  badge: { minWidth: 20, height: 20, borderRadius: 10, alignItems: "center", justifyContent: "center", paddingHorizontal: 5 }
});
