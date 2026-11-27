import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { useBlocks } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { spacing } from "../../theme/palette";
import { Button, EmptyState, ErrorState, Header, Loading, Screen, Text, toast } from "../../ui";
import { formatRelativeShort } from "../../utils/format";
import type { AppScreenProps } from "../../navigation/types";

export function BlockedUsersScreen({ navigation }: AppScreenProps<"BlockedUsers">) {
  const blocks = useBlocks();
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.blocks });
      toast("Personne débloquée.", "success");
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  return (
    <Screen>
      <Header title="Personnes bloquées" onBack={() => navigation.goBack()} />
      {blocks.isLoading ? (
        <Loading />
      ) : blocks.error ? (
        <ErrorState message={errorMessage(blocks.error)} onRetry={() => void blocks.refetch()} />
      ) : (blocks.data ?? []).length === 0 ? (
        <EmptyState icon="shield-checkmark-outline" title="Aucune personne bloquée" message="Tu peux bloquer quelqu'un depuis son profil ou une conversation." />
      ) : (
        <FlatList
          data={blocks.data}
          keyExtractor={(b) => b.userId}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <View style={styles.flex}>
                <Text variant="bodyStrong">{item.firstName || "Membre"}</Text>
                <Text variant="caption" tone="subtle">
                  Bloqué·e {formatRelativeShort(item.blockedAt)}
                </Text>
              </View>
              <Button label="Débloquer" variant="outline" size="sm" onPress={() => unblock.mutate(item.userId)} />
            </View>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm }
});
