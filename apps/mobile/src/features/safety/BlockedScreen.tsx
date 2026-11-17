import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { colors, spacing } from "../../theme";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text, showToast } from "../../ui";
import { shortTime } from "../../utils/format";

export function BlockedScreen() {
  const queryClient = useQueryClient();
  const blocks = useQuery({ queryKey: queryKeys.blocks, queryFn: async () => (await safetyApi.blocks()).blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      showToast("Membre débloqué.", "success");
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  if (blocks.isLoading)
    return (
      <Screen edges={["left", "right"]}>
        <LoadingState />
      </Screen>
    );
  if (blocks.isError)
    return (
      <Screen edges={["left", "right"]}>
        <ErrorState onRetry={() => void blocks.refetch()} />
      </Screen>
    );

  return (
    <Screen edges={["left", "right"]}>
      <FlatList
        data={blocks.data}
        keyExtractor={(b) => b.userId}
        ListEmptyComponent={
          <EmptyState
            icon="shield-checkmark-outline"
            title="Aucun membre bloqué"
            message="Les personnes que tu bloques apparaîtront ici."
          />
        }
        contentContainerStyle={styles.list}
        renderItem={({ item }) => (
          <View style={styles.row}>
            <View style={styles.flex}>
              <Text variant="bodyStrong">{item.firstName || "Membre"}</Text>
              <Text variant="caption" tone="subtle">
                Bloqué le {shortTime(item.blockedAt)}
              </Text>
            </View>
            <Button
              label="Débloquer"
              variant="secondary"
              compact
              loading={unblock.isPending && unblock.variables === item.userId}
              onPress={() => unblock.mutate(item.userId)}
            />
          </View>
        )}
      />
      <Text variant="caption" tone="subtle" style={styles.note}>
        Débloquer ne rétablit pas un match terminé.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    flexGrow: 1
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  flex: {
    flex: 1
  },
  note: {
    paddingVertical: spacing.md
  }
});
