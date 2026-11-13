import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { safetyApi } from "../../api/safety";
import type { RootScreenProps } from "../../navigation/types";
import { queryKeys } from "../../state/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, useToast } from "../../ui";
import { describeError } from "../../utils/errors";
import { formatRelative } from "../../utils/format";

export default function BlockedUsersScreen(_: RootScreenProps<"BlockedUsers">) {
  const { spacing } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const blocked = useQuery({ queryKey: queryKeys.blocked, queryFn: safetyApi.blocked });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      toast.show("Membre débloqué.", "success");
      void client.invalidateQueries({ queryKey: queryKeys.blocked });
    },
    onError: (e) => toast.show(describeError(e), "error")
  });

  return (
    <Screen testID="blocked-screen">
      <Header back title="Membres bloqués" />
      {blocked.isLoading ? (
        <LoadingState />
      ) : blocked.isError ? (
        <ErrorState message={describeError(blocked.error)} onAction={() => void blocked.refetch()} />
      ) : (
        <FlatList
          data={blocked.data}
          keyExtractor={(b) => b.userId}
          contentContainerStyle={{ gap: spacing.sm, flexGrow: 1 }}
          ListEmptyComponent={<EmptyState icon="shield-checkmark-outline" title="Personne n'est bloqué" />}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <Avatar name={item.firstName || "?"} size={44} />
              <View style={styles.flex}>
                <Text weight="medium">{item.firstName || "Membre"}</Text>
                <Text variant="caption" tone="subtle">
                  Bloqué·e {formatRelative(item.blockedAt).toLowerCase()}
                </Text>
              </View>
              <Button label="Débloquer" size="md" variant="secondary" onPress={() => unblock.mutate(item.userId)} />
            </View>
          )}
        />
      )}
      <Text variant="caption" tone="subtle" style={{ paddingVertical: spacing.sm }}>
        Débloquer ne restaure pas un match annulé.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: 12 }
});
