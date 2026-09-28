import React from "react";
import { FlatList, RefreshControl } from "react-native";
import { errorMessage } from "../api/client";
import type { BlockedUser } from "../api/types";
import { useBlockedUsers, useUnblock } from "../hooks/useSafety";
import type { MainScreenProps } from "../navigation/types";
import { EmptyView, ErrorView, ListSkeleton, showToast } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, ListItem } from "../shared/ui";
import { useTheme } from "../theme";

export default function BlockedUsersScreen(_props: MainScreenProps<"BlockedUsers">) {
  const theme = useTheme();
  const query = useBlockedUsers();
  const unblock = useUnblock();

  const renderItem = ({ item }: { item: BlockedUser }) => (
    <ListItem
      right={
        <Button
          a11yLabel={`Unblock ${item.firstName}`}
          label="Unblock"
          loading={unblock.isPending && unblock.variables === item.userId}
          onPress={() =>
            unblock.mutate(item.userId, {
              onSuccess: () => showToast(`${item.firstName} was unblocked.`, "success"),
              onError: (error) => showToast(errorMessage(error), "error")
            })
          }
          variant="secondary"
        />
      }
      subtitle={`Blocked ${new Date(item.blockedAt).toLocaleDateString()}`}
      title={item.firstName}
    />
  );

  return (
    <SafeAreaLayout edges={["bottom"]}>
      {query.isPending ? (
        <ListSkeleton rows={4} />
      ) : query.isError ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={query.data.length === 0 ? { flexGrow: 1 } : undefined}
          data={query.data}
          keyExtractor={(item) => item.userId}
          ListEmptyComponent={
            <EmptyView message="People you block cannot see you and you will not see them." title="No one is blocked" />
          }
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching}
              tintColor={theme.primary}
            />
          }
          renderItem={renderItem}
        />
      )}
    </SafeAreaLayout>
  );
}
