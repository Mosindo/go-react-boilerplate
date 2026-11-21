import React from "react";
import { FlatList, View } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { makeStyles } from "../../theme";
import { Avatar, Button, confirm, EmptyState, ErrorState, Header, LoadingState, Screen, showToast, Text } from "../../ui";

export function BlockedUsersScreen() {
  const styles = useStyles();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: queryKeys.blocks, queryFn: async () => (await safetyApi.blocks()).blocks });

  const unblock = async (userId: string, name: string) => {
    if (
      !(await confirm({
        title: `Débloquer ${name} ?`,
        message: "Vous pourrez de nouveau vous croiser dans la découverte si vos critères correspondent."
      }))
    )
      return;
    try {
      await safetyApi.unblock(userId);
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen padded={false}>
      <Header title="Personnes bloquées" />
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          data={query.data}
          keyExtractor={(b) => b.user.id}
          contentContainerStyle={styles.list}
          ListEmptyComponent={
            <EmptyState
              icon="shield-checkmark-outline"
              title="Aucun blocage"
              message="Les personnes que vous bloquez ne peuvent plus vous voir ni vous écrire."
            />
          }
          renderItem={({ item }) => (
            <View style={styles.row}>
              <Avatar uri={item.user.photoUrl} name={item.user.firstName} size={44} />
              <Text variant="bodyStrong" style={styles.name}>
                {item.user.firstName}
              </Text>
              <Button label="Débloquer" variant="secondary" size="sm" onPress={() => void unblock(item.user.id, item.user.firstName)} />
            </View>
          )}
        />
      )}
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  list: { flexGrow: 1, paddingHorizontal: t.spacing.xl },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.md,
    paddingVertical: t.spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: t.colors.border
  },
  name: { flex: 1 }
}));
