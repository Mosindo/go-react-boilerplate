import { useQuery, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, StyleSheet, Text, View } from "react-native";
import { fetchBlocked, unblockUser } from "../../api/safety";
import { Avatar } from "../../components/Avatar";
import { Button } from "../../components/Button";
import { EmptyView, ErrorView, LoadingView } from "../../components/StateViews";
import { queryKeys } from "../../hooks/queries";
import { errorMessage } from "../../hooks/useAsyncAction";
import { bumpDiscoverVersion } from "../../state/discoverStore";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

export default function BlockedScreen() {
  const theme = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: queryKeys.blocked, queryFn: fetchBlocked });

  if (query.isLoading) return <LoadingView />;
  if (query.isError)
    return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const blocks = query.data ?? [];
  if (blocks.length === 0) {
    return (
      <EmptyView
        icon="ban-outline"
        message="Les personnes que vous bloquez apparaissent ici."
        title="Aucun utilisateur bloqué"
      />
    );
  }

  return (
    <FlatList
      contentContainerStyle={styles.content}
      data={blocks}
      keyExtractor={(b) => b.userId}
      renderItem={({ item }) => (
        <View style={[styles.row, { borderBottomColor: theme.border }]}>
          <Avatar name={item.firstName || "?"} size={44} />
          <Text style={[styles.name, { color: theme.text }]}>
            {item.firstName || "Utilisateur"}
          </Text>
          <Button
            onPress={async () => {
              try {
                await unblockUser(item.userId);
                bumpDiscoverVersion();
                await queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
                toast.show("Utilisateur débloqué.", "success");
              } catch (error) {
                toast.show(errorMessage(error), "error");
              }
            }}
            style={styles.button}
            title="Débloquer"
            variant="secondary"
          />
        </View>
      )}
      style={{ backgroundColor: theme.background }}
    />
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg },
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  name: { flex: 1, marginLeft: spacing.md, fontSize: fontSizes.md, fontWeight: "600" },
  button: { minHeight: 40, paddingHorizontal: spacing.lg }
});
