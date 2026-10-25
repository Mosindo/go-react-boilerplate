import React, { useCallback, useMemo } from "react";
import { FlatList, StyleSheet } from "react-native";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys, safetyApi, type BlockedUser } from "../../api";
import type { Paged } from "../../realtime/cache";
import { showToast } from "../../shared/feedback";
import { flattenPages, cursorQuery } from "../../shared/pagination";
import { Avatar, Button, EmptyState, ErrorState, ListRow, Screen, Skeleton } from "../../shared/ui";
import { spacing } from "../../theme";

const blocksQuery = cursorQuery(queryKeys.blocks, safetyApi.blocks);

const BlockedRow = React.memo(function BlockedRow({
  item,
  onUnblock,
  busy
}: {
  item: BlockedUser;
  onUnblock: (userId: string) => void;
  busy: boolean;
}) {
  return (
    <ListRow
      left={<Avatar name={item.firstName} size={48} thumbUrl={item.thumbUrl} />}
      right={<Button disabled={busy} label="Débloquer" onPress={() => onUnblock(item.userId)} variant="secondary" />}
      title={item.firstName}
    />
  );
});

export function BlockedUsersScreen() {
  const queryClient = useQueryClient();
  const query = useInfiniteQuery(blocksQuery);
  const items = useMemo(() => flattenPages(query.data), [query.data]);

  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: (_void, userId) => {
      queryClient.setQueryData<Paged<BlockedUser>>(queryKeys.blocks, (data) =>
        data
          ? {
              ...data,
              pages: data.pages.map((page) => ({ ...page, items: page.items.filter((item) => item.userId !== userId) }))
            }
          : data
      );
      showToast("Utilisateur débloqué.");
    }
  });
  const { mutate: unblockUser, isPending } = unblock;

  const renderItem = useCallback(
    ({ item }: { item: BlockedUser }) => <BlockedRow busy={isPending} item={item} onUnblock={unblockUser} />,
    [isPending, unblockUser]
  );

  if (query.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  if (query.isLoading) {
    return (
      <Screen>
        <Skeleton height={48} />
      </Screen>
    );
  }

  return (
    <Screen padded={false} testID="blocked-screen">
      {items.length === 0 ? (
        <EmptyState
          icon="shield-checkmark-outline"
          message="Les personnes que vous bloquez apparaîtront ici."
          title="Aucun utilisateur bloqué"
        />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(item) => item.userId}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          renderItem={renderItem}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({ list: { paddingHorizontal: spacing.xl, paddingTop: spacing.md } });
