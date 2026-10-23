import React, { useCallback, useState } from "react";
import { ActivityIndicator, FlatList, RefreshControl, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { MatchSummary } from "../../api/types";
import { useModeration } from "../../features/useModeration";
import { t } from "../../i18n/fr";
import { formatListTime } from "../../lib/dates";
import { messagePreview } from "../../lib/format";
import { useMatches } from "../../queries";
import { useSession } from "../../session/SessionProvider";
import { createStyles, useTheme } from "../../theme/ThemeProvider";
import {
  CountBadge,
  EmptyState,
  ErrorState,
  ListSkeleton,
  Photo,
  Screen,
  ScalePressable,
  Text
} from "../../ui";
import type { MainStackParams, TabParams } from "../navTypes";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Matches">,
  NativeStackScreenProps<MainStackParams>
>;

const useStyles = createStyles(({ colors, spacing }) => ({
  row: {
    minHeight: 72,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border
  },
  avatar: { width: 56, height: 56, borderRadius: 28 },
  main: { flex: 1, gap: 2 },
  top: { flexDirection: "row", justifyContent: "space-between", gap: spacing.sm },
  side: { alignItems: "flex-end", gap: spacing.xs },
  more: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  footer: { paddingVertical: spacing.lg }
}));

export default function MatchesScreen({ navigation }: Props) {
  const styles = useStyles();
  const { colors } = useTheme();
  const { userId } = useSession();
  const matches = useMatches();
  const moderation = useModeration();
  const [refreshing, setRefreshing] = useState(false);

  const items = matches.data?.pages.flatMap((page) => page.matches) ?? [];

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await matches.refetch();
    } finally {
      setRefreshing(false);
    }
  }, [matches]);

  const openChat = (match: MatchSummary) =>
    navigation.navigate("Chat", {
      matchId: match.id,
      userId: match.user.userId,
      firstName: match.user.firstName,
      thumbUrl: match.user.thumbUrl
    });

  const openMenu = (match: MatchSummary) =>
    moderation.openMenu(
      { matchId: match.id, userId: match.user.userId, firstName: match.user.firstName },
      {
        title: t.matches.menuTitle(match.user.firstName),
        withHide: true,
        onViewProfile: () => navigation.navigate("ProfileDetail", { userId: match.user.userId })
      }
    );

  if (matches.isPending) {
    return (
      <Screen testID="matches-screen">
        <ListSkeleton />
      </Screen>
    );
  }
  if (matches.isError && items.length === 0) {
    return (
      <Screen testID="matches-screen">
        <ErrorState onRetry={() => matches.refetch()} title={t.matches.errorTitle} />
      </Screen>
    );
  }

  return (
    <Screen testID="matches-screen">
      <FlatList
        contentContainerStyle={items.length === 0 ? { flexGrow: 1 } : undefined}
        data={items}
        keyExtractor={(match) => match.id}
        ListEmptyComponent={
          <EmptyState
            actionLabel={t.matches.emptyCta}
            glyph="♡"
            message={t.matches.emptyText}
            onAction={() => navigation.navigate("Discover")}
            testID="matches-empty"
            title={t.matches.emptyTitle}
          />
        }
        ListFooterComponent={
          matches.isFetchingNextPage ? (
            <View style={styles.footer}>
              <ActivityIndicator color={colors.primary} />
            </View>
          ) : null
        }
        onEndReached={() => {
          if (matches.hasNextPage && !matches.isFetchingNextPage) void matches.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        refreshControl={
          <RefreshControl onRefresh={refresh} refreshing={refreshing} tintColor={colors.primary} />
        }
        renderItem={({ item }) => (
          <View style={styles.row}>
            <ScalePressable
              accessibilityHint={t.matches.menuTitle(item.user.firstName)}
              accessibilityLabel={`${t.matches.open(item.user.firstName)}${
                item.unreadCount > 0 ? `, ${t.matches.unread(item.unreadCount)}` : ""
              }`}
              accessibilityRole="button"
              onLongPress={() => openMenu(item)}
              onPress={() => openChat(item)}
              style={{ flex: 1 }}
              testID={`match-row-${item.user.firstName}`}
            >
              <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
                <Photo name={item.user.firstName} path={item.user.thumbUrl} style={styles.avatar} />
                <View style={styles.main}>
                  <View style={styles.top}>
                    <Text numberOfLines={1} style={{ flex: 1 }} variant="label">
                      {item.user.firstName}
                    </Text>
                    <Text tone="muted" variant="caption">
                      {formatListTime(item.lastMessage?.createdAt ?? item.lastActivityAt)}
                    </Text>
                  </View>
                  <View style={styles.top}>
                    <Text
                      numberOfLines={1}
                      style={{ flex: 1, fontWeight: item.unreadCount > 0 ? "700" : "400" }}
                      tone={item.lastMessage ? "default" : "muted"}
                    >
                      {messagePreview(item.lastMessage, userId)}
                    </Text>
                    <CountBadge count={item.unreadCount} />
                  </View>
                </View>
              </View>
            </ScalePressable>
            <ScalePressable
              accessibilityLabel={`${t.common.more} : ${item.user.firstName}`}
              accessibilityRole="button"
              onPress={() => openMenu(item)}
              testID={`match-menu-${item.user.firstName}`}
            >
              <View style={styles.more}>
                <Text tone="muted" variant="heading">
                  ⋯
                </Text>
              </View>
            </ScalePressable>
          </View>
        )}
      />
    </Screen>
  );
}
