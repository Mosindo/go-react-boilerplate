import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useBlocks, useSafetyMutations } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme/tokens";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";

export default function BlockedUsersScreen({ navigation }: AppScreenProps<"BlockedUsers">) {
  const blocks = useBlocks();
  const { unblock } = useSafetyMutations();
  return (
    <Screen>
      <Header title="Personnes bloquées" onBack={navigation.goBack} />
      {blocks.isLoading ? (
        <LoadingState />
      ) : blocks.error ? (
        <ErrorState message={errorMessage(blocks.error)} onRetry={() => blocks.refetch()} />
      ) : (
        <FlatList
          data={blocks.data}
          keyExtractor={(b) => b.userId}
          ListEmptyComponent={<EmptyState icon="shield-checkmark-outline" title="Personne n'est bloqué" />}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <Text variant="bodyStrong" style={styles.flex}>
                {item.firstName || "Compte supprimé"}
              </Text>
              <Button
                label="Débloquer"
                variant="secondary"
                compact
                loading={unblock.isPending && unblock.variables === item.userId}
                onPress={async () => {
                  try {
                    await unblock.mutateAsync(item.userId);
                    showToast(`${item.firstName} est débloqué·e`, "success");
                  } catch (err) {
                    showToast(errorMessage(err), "error");
                  }
                }}
              />
            </View>
          )}
        />
      )}
      <Text variant="caption" tone="subtle" style={styles.note}>
        Débloquer ne rétablit pas un match terminé.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  flex: { flex: 1 },
  note: { paddingVertical: spacing.lg }
});
