import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { FlatList } from "react-native";
import { unblockUser } from "../api/platform";
import { useBlocks } from "../hooks/queries";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../realtime/cache";
import { EmptyView, ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Avatar, Button, ListItem } from "../shared/ui";

export default function BlockedScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useBlocks();

  const unblock = async (userId: string, name: string) => {
    try {
      await unblockUser(userId);
      await queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      showSuccess(`${name} est débloqué(e).`);
    } catch (e) {
      showError(errorMessage(e));
    }
  };

  return (
    <ScreenContainer testID="blocked-screen">
      <Header title="Personnes bloquées" onBack={() => navigation.goBack()} />
      {isLoading ? <LoadingView /> : null}
      {isError ? <ErrorView error={error} onRetry={() => void refetch()} /> : null}
      {data && data.length === 0 ? <EmptyView icon="shield-checkmark-outline" title="Personne n'est bloqué" message="Les personnes que vous bloquez apparaissent ici. Elles ne sont jamais prévenues." /> : null}
      {data && data.length > 0 ? (
        <FlatList
          data={data}
          keyExtractor={(b) => b.user.id}
          renderItem={({ item }) => (
            <ListItem
              leading={<Avatar name={item.user.firstName} path={item.user.photos[0]?.url} />}
              title={item.user.firstName}
              trailing={<Button label="Débloquer" variant="secondary" compact onPress={() => void unblock(item.user.id, item.user.firstName)} />}
            />
          )}
        />
      ) : null}
    </ScreenContainer>
  );
}
