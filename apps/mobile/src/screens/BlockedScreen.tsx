import React from "react";
import { FlatList } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { mediaUrl } from "../api/client";
import { unblockUser } from "../api/platform";
import { queryKeys, useBlocked } from "../hooks/queries";
import { errorMessage } from "../i18n/errors";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Avatar, Button, ListItem } from "../shared/ui";
import { useTheme } from "../theme";

export default function BlockedScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useBlocked();

  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
      showSuccess("Utilisateur débloqué.");
    },
    onError: (error) => showError(errorMessage(error))
  });

  return (
    <ScreenContainer testID="blocked-screen">
      <Header onBack={() => navigation.goBack()} title="Utilisateurs bloqués" />
      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      ) : (query.data ?? []).length === 0 ? (
        <EmptyView
          icon={<Ionicons color={palette.textSubtle} name="shield-checkmark-outline" size={56} />}
          message="Les personnes que vous bloquez ne peuvent plus vous voir ni vous écrire."
          style={{ flex: 1 }}
          title="Personne n'est bloqué"
        />
      ) : (
        <FlatList
          data={query.data}
          keyExtractor={(b) => b.userId}
          renderItem={({ item }) => (
            <ListItem
              bordered
              leading={<Avatar name={item.firstName} uri={mediaUrl(item.thumbUrl)} />}
              title={item.firstName || "Compte supprimé"}
              trailing={
                <Button
                  disabled={unblock.isPending}
                  label="Débloquer"
                  onPress={() => unblock.mutate(item.userId)}
                  size="sm"
                  variant="secondary"
                />
              }
            />
          )}
        />
      )}
    </ScreenContainer>
  );
}
