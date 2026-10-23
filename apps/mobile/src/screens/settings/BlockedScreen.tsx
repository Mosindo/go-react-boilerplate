import React from "react";
import { FlatList, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { unblockUser } from "../../api";
import type { BlockedUser } from "../../api/types";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { useBlocks } from "../../queries";
import { queryKeys } from "../../session/queryKeys";
import { createStyles } from "../../theme/ThemeProvider";
import { Button, EmptyState, ErrorState, ListSkeleton, Screen, Text, useToast } from "../../ui";

const useStyles = createStyles(({ colors, spacing }) => ({
  row: {
    minHeight: 64,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border
  }
}));

export default function BlockedScreen() {
  const styles = useStyles();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useBlocks();

  const unblock = useMutation({
    mutationFn: (user: BlockedUser) => unblockUser(user.userId),
    onSuccess: (_r, user) => {
      queryClient.setQueryData<BlockedUser[]>(queryKeys.blocks, (list) =>
        list?.filter((item) => item.userId !== user.userId)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      toast.success(t.blocked.unblocked);
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  if (query.isPending) {
    return (
      <Screen>
        <ListSkeleton rows={4} />
      </Screen>
    );
  }
  if (query.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => query.refetch()} title={t.errors.generic} />
      </Screen>
    );
  }

  return (
    <Screen testID="blocked-screen">
      <FlatList
        contentContainerStyle={query.data.length === 0 ? { flexGrow: 1 } : undefined}
        data={query.data}
        keyExtractor={(user) => user.userId}
        ListEmptyComponent={
          <EmptyState glyph="☺" message={t.blocked.emptyText} title={t.blocked.emptyTitle} />
        }
        ListFooterComponent={
          query.data.length > 0 ? (
            <Text tone="muted" variant="caption" style={{ paddingVertical: 12 }}>
              {t.blocked.note}
            </Text>
          ) : null
        }
        renderItem={({ item }) => (
          <View style={styles.row}>
            <Text style={{ flex: 1 }} variant="label">
              {item.firstName}
            </Text>
            <View style={{ width: 130 }}>
              <Button
                accessibilityLabel={t.blocked.unblockLabel(item.firstName)}
                compact
                label={t.blocked.unblock}
                loading={unblock.isPending && unblock.variables?.userId === item.userId}
                onPress={() => unblock.mutate(item)}
                testID={`unblock-${item.firstName}`}
                variant="secondary"
              />
            </View>
          </View>
        )}
      />
    </Screen>
  );
}
