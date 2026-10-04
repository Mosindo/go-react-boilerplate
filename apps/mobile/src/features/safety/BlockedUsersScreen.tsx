import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { getErrorMessage } from "../../api/client";
import { useBlocks, useUnblockUser } from "../../hooks/useSafety";
import { spacing } from "../../theme";
import { Button, EmptyState, ErrorState, ListSkeleton, Text, toast } from "../../ui";

export function BlockedUsersScreen() {
  const blocks = useBlocks();
  const unblock = useUnblockUser();

  if (blocks.isPending) return <ListSkeleton rows={4} />;
  if (blocks.isError) return <ErrorState error={blocks.error} onRetry={() => void blocks.refetch()} />;
  if (blocks.data.length === 0) {
    return (
      <EmptyState
        icon="shield-checkmark-outline"
        message="People you block won't see you or be able to contact you."
        title="No one is blocked"
      />
    );
  }
  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={blocks.data}
      keyExtractor={(item) => item.userId}
      onRefresh={() => void blocks.refetch()}
      refreshing={blocks.isRefetching}
      renderItem={({ item }) => (
        <View style={styles.row}>
          <View style={styles.name}>
            <Text weight="semibold">{item.displayName}</Text>
            <Text tone="muted" variant="caption">
              Blocked {new Date(item.blockedAt).toLocaleDateString()}
            </Text>
          </View>
          <Button
            accessibilityLabel={`Unblock ${item.displayName}`}
            fullWidth={false}
            label="Unblock"
            loading={unblock.isPending && unblock.variables === item.userId}
            onPress={() =>
              unblock.mutate(item.userId, {
                onSuccess: () => toast.success(`${item.displayName} was unblocked`),
                onError: (e) => toast.error(getErrorMessage(e))
              })
            }
            variant="secondary"
          />
        </View>
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.md },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 64 },
  name: { flex: 1 }
});
