import React from "react";
import { FlatList, StyleSheet } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { unblockUser } from "../../api/dating";
import { qk, useBlocks } from "../../hooks/queries";
import type { MainScreenProps } from "../../navigation/types";
import { Avatar, Button, EmptyState, ErrorState, Header, ListRow, LoadingState, Screen, showToast, spacing } from "../../ui";

export default function BlockedUsersScreen({ navigation }: MainScreenProps<"BlockedUsers">) {
  const client = useQueryClient();
  const blocks = useBlocks();

  const unblock = async (userId: string, name: string) => {
    try {
      await unblockUser(userId);
      await client.invalidateQueries({ queryKey: qk.blocks });
      showToast(`${name} est débloqué·e.`);
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen>
      <Header onBack={navigation.goBack} title="Personnes bloquées" />
      {blocks.isPending ? (
        <LoadingState />
      ) : blocks.isError ? (
        <ErrorState message={errorMessage(blocks.error)} onAction={() => void blocks.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={blocks.data.length ? styles.list : styles.empty}
          data={blocks.data}
          keyExtractor={(b) => b.userId}
          ListEmptyComponent={
            <EmptyState
              icon="shield-checkmark-outline"
              message="Les personnes que vous bloquez ne peuvent plus vous voir ni vous écrire."
              title="Personne n'est bloqué"
            />
          }
          renderItem={({ item }) => (
            <ListRow
              leading={<Avatar name={item.firstName} size={40} url={item.photo?.url} />}
              title={item.firstName}
              trailing={<Button compact label="Débloquer" onPress={() => void unblock(item.userId, item.firstName)} variant="secondary" />}
            />
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.xs
  },
  empty: {
    flexGrow: 1
  }
});
