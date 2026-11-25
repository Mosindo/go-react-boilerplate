import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/safety";
import { queryKeys } from "../../hooks/queryClient";
import { useBlocks } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, showToast } from "../../ui";
import { formatRelative } from "../../utils/format";

export function BlockedUsersScreen({ navigation }: AppScreenProps<"BlockedUsers">) {
  const qc = useQueryClient();
  const blocks = useBlocks();
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.blocks });
      void qc.invalidateQueries({ queryKey: queryKeys.discover });
      showToast("Personne débloquée.", "success");
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  return (
    <Screen testID="blocked-screen">
      <Header title="Personnes bloquées" onBack={navigation.goBack} />
      {blocks.isLoading ? (
        <LoadingState />
      ) : blocks.error ? (
        <ErrorState message={errorMessage(blocks.error)} onRetry={() => void blocks.refetch()} />
      ) : (blocks.data ?? []).length === 0 ? (
        <EmptyState
          icon="shield-checkmark-outline"
          title="Aucun blocage"
          body="Les personnes que vous bloquez ne peuvent plus vous voir ni vous écrire."
        />
      ) : (
        <FlatList
          data={blocks.data}
          keyExtractor={(b) => b.userId}
          contentContainerStyle={styles.list}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <View style={styles.flex}>
                <Text variant="bodyStrong">{item.firstName || "Membre"}</Text>
                <Text variant="caption" color="textSubtle">
                  Bloqué·e {formatRelative(item.blockedAt)}
                </Text>
              </View>
              <Button label="Débloquer" variant="secondary" compact onPress={() => unblock.mutate(item.userId)} />
            </View>
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
  row: {
    flexDirection: "row",
    alignItems: "center",
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface,
    gap: spacing.sm
  },
  flex: {
    flex: 1
  }
});
