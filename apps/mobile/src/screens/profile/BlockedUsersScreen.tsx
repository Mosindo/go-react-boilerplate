import React from "react";
import { Alert, FlatList } from "react-native";
import { resolveMediaUrl } from "../../api/client";
import type { UserSummary } from "../../api/types";
import { useBlockedUsers, useUnblock } from "../../hooks/useProfile";
import { strings } from "../../i18n/strings";
import { EmptyView, ErrorView, LoadingView, reportError, showToast } from "../../shared/feedback";
import { SafeAreaLayout } from "../../shared/layout";
import { Avatar, Button, ListItem, spacing } from "../../shared/ui";

export default function BlockedUsersScreen() {
  const query = useBlockedUsers();
  const unblock = useUnblock();

  function confirm(user: UserSummary) {
    Alert.alert(strings.blocked.unblockTitle(user.firstName), strings.blocked.unblockBody, [
      { text: strings.common.cancel, style: "cancel" },
      {
        text: strings.blocked.unblock,
        onPress: () =>
          unblock.mutate(user.userId, {
            onSuccess: () => showToast(strings.blocked.unblocked(user.firstName), "success"),
            onError: (error) => reportError(error)
          })
      }
    ]);
  }

  return (
    <SafeAreaLayout edges={["left", "right", "bottom"]}>
      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView
          message={strings.errors.generic}
          onAction={() => void query.refetch()}
          style={{ margin: spacing.lg }}
        />
      ) : (
        <FlatList
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm, flexGrow: 1 }}
          data={query.data}
          keyExtractor={(item) => item.userId}
          ListEmptyComponent={
            <EmptyView message={strings.blocked.emptyBody} title={strings.blocked.emptyTitle} />
          }
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching}
          renderItem={({ item }) => (
            <ListItem
              leading={
                <Avatar name={item.firstName} uri={item.photo ? resolveMediaUrl(item.photo.url) : null} />
              }
              subtitle={strings.blocked.age(item.age)}
              title={item.firstName}
              trailing={
                <Button
                  disabled={unblock.isPending}
                  label={strings.blocked.unblock}
                  onPress={() => confirm(item)}
                  size="sm"
                  variant="outline"
                />
              }
            />
          )}
        />
      )}
    </SafeAreaLayout>
  );
}
