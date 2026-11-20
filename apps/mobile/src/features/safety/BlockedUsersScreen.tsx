import React from "react";
import { FlatList, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/endpoints";
import { errorMessage, formatListTime } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, showToast, Text } from "../../ui";

export function BlockedUsersScreen() {
  const { colors, spacing } = useTheme();
  const queryClient = useQueryClient();
  const blocked = useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.blocked });
      showToast("Personne débloquée.", "success");
    },
    onError: (e) => showToast(errorMessage(e), "error")
  });

  return (
    <Screen>
      <Header back title="Personnes bloquées" />
      {blocked.isLoading ? (
        <LoadingState />
      ) : blocked.isError ? (
        <ErrorState message={errorMessage(blocked.error)} onRetry={() => void blocked.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm, flexGrow: 1 }}
          data={blocked.data}
          keyExtractor={(b) => b.userId}
          ListEmptyComponent={<EmptyState icon="shield-checkmark-outline" message="Tu n’as bloqué personne." title="Aucun blocage" />}
          renderItem={({ item }) => (
            <View
              style={{ flexDirection: "row", alignItems: "center", padding: spacing.md, borderRadius: 14, backgroundColor: colors.surface, gap: spacing.sm }}
            >
              <View style={{ flex: 1 }}>
                <Text variant="bodyStrong">{item.firstName || "Membre"}</Text>
                <Text tone="subtle" variant="caption">
                  Bloqué·e le {formatListTime(item.blockedAt)}
                </Text>
              </View>
              <Button label="Débloquer" onPress={() => unblock.mutate(item.userId)} size="sm" variant="secondary" />
            </View>
          )}
        />
      )}
    </Screen>
  );
}
