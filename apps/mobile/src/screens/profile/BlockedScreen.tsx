import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { unblockUser } from "../../api/safety";
import { useBlocked } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, showToast } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "Blocked">;

export function BlockedScreen({ navigation }: Props) {
  const blocked = useBlocked();
  const unblock = async (userId: string, name: string) => {
    try {
      await unblockUser(userId);
      await blocked.refetch();
      showToast(`${name || "Ce membre"} est débloqué·e.`, "success");
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };
  return (
    <Screen>
      <Header onBack={navigation.goBack} title="Personnes bloquées" />
      {blocked.isLoading ? (
        <LoadingState />
      ) : blocked.isError ? (
        <ErrorState message={errorMessage(blocked.error)} onRetry={() => void blocked.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={blocked.data ?? []}
          keyExtractor={(b) => b.userId}
          ListEmptyComponent={<EmptyState icon="happy-outline" message="Vous n'avez bloqué personne." title="Liste vide" />}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <Text style={styles.name} variant="bodyStrong">
                {item.firstName || "Membre"}
              </Text>
              <Button label="Débloquer" onPress={() => void unblock(item.userId, item.firstName)} size="md" variant="secondary" />
            </View>
          )}
        />
      )}
      <Text style={styles.note} tone="subtle" variant="caption">
        Débloquer ne recrée pas un match supprimé.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { flexGrow: 1, gap: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", paddingVertical: spacing.xs },
  name: { flex: 1 },
  note: { paddingVertical: spacing.md }
});
