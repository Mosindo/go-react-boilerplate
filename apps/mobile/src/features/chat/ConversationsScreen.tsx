import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { Conversation } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { useConversations } from "../../state/queries";
import { radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { shortTime } from "../../utils/format";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function ConversationsScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const { user } = useAuth();
  const { data, isLoading, isError, error, refetch, isRefetching } = useConversations();
  const [fresh, threads] = useMemo(() => {
    const list = data ?? [];
    return [list.filter((c) => !c.lastMessage), list.filter((c) => c.lastMessage)];
  }, [data]);

  const open = (c: Conversation) => navigation.navigate("Conversation", { matchId: c.matchId, user: c.user });

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={refetch} />;

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <Text accessibilityRole="header" variant="title">
          Matchs
        </Text>
      </View>
      {!data?.length ? (
        <EmptyState
          actionLabel="Découvrir des profils"
          body="Quand une personne que vous aimez vous aime en retour, elle apparaîtra ici."
          icon="heart-outline"
          onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshWrap}>
                <Text style={styles.sectionTitle} tone="subtle" variant="overline">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.fresh} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((c) => (
                    <Pressable accessibilityLabel={`Écrire à ${c.user.firstName}`} accessibilityRole="button" key={c.matchId} onPress={() => open(c)} style={styles.freshItem}>
                      <View style={[styles.freshRing, { borderColor: colors.primary }]}>
                        <Avatar photo={c.user.photos[0]} size={68} />
                      </View>
                      <Text numberOfLines={1} variant="caption" weight="600">
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {threads.length ? (
                  <Text style={styles.sectionTitle} tone="subtle" variant="overline">
                    Messages
                  </Text>
                ) : null}
              </View>
            ) : (
              <Text style={styles.sectionTitle} tone="subtle" variant="overline">
                Messages
              </Text>
            )
          }
          data={threads}
          keyExtractor={(c) => c.matchId}
          refreshControl={<RefreshControl onRefresh={refetch} refreshing={isRefetching} tintColor={colors.primary} />}
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            const unread = item.unreadCount > 0;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, pressed ? { backgroundColor: colors.surfaceMuted } : null]}
                testID={`conversation-${item.user.firstName}`}
              >
                <Avatar photo={item.user.photos[0]} size={56} />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} variant="heading">
                      {item.user.firstName}
                    </Text>
                    {last ? (
                      <Text tone={unread ? "primary" : "subtle"} variant="caption">
                        {shortTime(last.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} tone={unread ? "default" : "muted"} weight={unread ? "600" : undefined}>
                      {mine ? "Vous : " : ""}
                      {last?.body}
                    </Text>
                    {unread ? (
                      <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                        <Text tone="onPrimary" variant="caption" weight="700">
                          {item.unreadCount > 9 ? "9+" : item.unreadCount}
                        </Text>
                      </View>
                    ) : null}
                  </View>
                </View>
              </Pressable>
            );
          }}
          ListEmptyComponent={
            <Text center style={styles.hint} tone="muted">
              Lancez la conversation avec vos nouveaux matchs !
            </Text>
          }
          contentContainerStyle={styles.list}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingTop: spacing.md, paddingBottom: spacing.xs },
  list: { paddingBottom: spacing.xxl },
  freshWrap: { gap: spacing.xs },
  sectionTitle: { paddingHorizontal: spacing.lg, marginTop: spacing.sm, marginBottom: spacing.xs },
  fresh: { paddingHorizontal: spacing.lg, gap: spacing.md },
  freshItem: { alignItems: "center", gap: 6, width: 78 },
  freshRing: { borderWidth: 2, borderRadius: 40, padding: 2 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  flex: { flex: 1 },
  badge: { minWidth: 22, height: 22, borderRadius: radii.pill, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 },
  hint: { padding: spacing.xl }
});
