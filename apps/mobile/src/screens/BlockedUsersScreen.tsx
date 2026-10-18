import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { unblockUser } from "../api/safety";
import { NavHeader } from "../components/NavHeader";
import { Screen } from "../components/Screen";
import { queryKeys, useBlocks } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, showToast } from "../shared/feedback";
import { Button, Loader, Text, colors, radii, spacing } from "../shared/ui";
import { formatLongDate } from "../utils/format";

type Props = NativeStackScreenProps<MainStackParamList, "BlockedUsers">;

export default function BlockedUsersScreen({ navigation }: Props) {
  const client = useQueryClient();
  const query = useBlocks();
  const unblock = useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: () => {
      showToast("Utilisateur débloqué.", "success");
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  return (
    <Screen contentStyle={styles.content} header={<NavHeader onBack={() => navigation.goBack()} title="Personnes bloquées" />} testID="blocked-screen">
      {query.isLoading ? <Loader fullScreen label="Chargement…" /> : null}
      {query.isError ? <ErrorView actionLabel="Réessayer" message="Impossible de charger la liste." onAction={() => void query.refetch()} title="Oups" /> : null}
      {query.data ? (
        <FlatList
          ListEmptyComponent={<EmptyView message="Les personnes que vous bloquez apparaissent ici." testID="blocked-empty" title="Aucune personne bloquée" />}
          contentContainerStyle={styles.list}
          data={query.data}
          keyExtractor={(b) => b.userId}
          renderItem={({ item }) => (
            <View style={styles.row}>
              <View style={styles.copy}>
                <Text weight="semibold">{item.firstName || "Utilisateur"}</Text>
                <Text tone="muted" variant="caption">
                  Bloqué(e) le {formatLongDate(item.blockedAt)}
                </Text>
              </View>
              <Button label="Débloquer" loading={unblock.isPending && unblock.variables === item.userId} onPress={() => unblock.mutate(item.userId)} size="sm" testID={`unblock-${item.userId}`} variant="outline" />
            </View>
          )}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: 0 },
  list: { padding: spacing.lg, gap: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, borderWidth: 1, borderColor: colors.border, backgroundColor: colors.backgroundElevated },
  copy: { flex: 1, gap: 2 }
});
